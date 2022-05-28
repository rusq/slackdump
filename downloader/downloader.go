package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
)

const (
	defDownloadDir = "." // default download directory is current.
	defRetries     = 3   // default number of retries if download fails
	defNumWorkers  = 4   // number of download processes
	defLimit       = 5000
	defFileBufSz   = 100
)

type Client struct {
	client  Downloader
	limiter *rate.Limiter
	fs      fsadapter.FS

	retries int
	workers int

	mu           sync.Mutex // mutex prevents race condition when starting/stopping
	fileRequests chan FileRequest
	wg           *sync.WaitGroup
	started      bool
}

// Downloader is the file downloader interface.  It exists primarily for mocking
// in tests.
type Downloader interface {
	// GetFile retreives a given file from its private download URL
	GetFile(downloadURL string, writer io.Writer) error
}

type Option func(*Client)

// Limiter uses the initialised limiter instead of built in.
func Limiter(l *rate.Limiter) Option {
	return func(c *Client) {
		if l != nil {
			c.limiter = l
		}
	}
}

func Retries(n int) Option {
	return func(c *Client) {
		if n <= 0 {
			n = defRetries
		}
		c.retries = n
	}
}

func Workers(n int) Option {
	return func(c *Client) {
		if n <= 0 {
			n = defNumWorkers
		}
		c.workers = n
	}
}

// Ne initialises new file downloader
func New(client Downloader, fs fsadapter.FS, opts ...Option) *Client {
	if client == nil {
		// better safe than sorry
		panic("programming error: client is nil")
	}
	c := &Client{
		client:  client,
		fs:      fs,
		limiter: rate.NewLimiter(defLimit, 1),
		retries: defRetries,
		workers: defNumWorkers,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SaveFile saves a single file to the specified directory synchrounously.
func (c *Client) SaveFile(ctx context.Context, dir string, f *slack.File) (int64, error) {
	return c.saveFile(ctx, dir, f)
}

type FileRequest struct {
	Directory string
	File      *slack.File
}

// Start starts an async file downloader.  If the downloader
// is already started, it does nothing.
func (c *Client) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		// already started
		return
	}
	req := make(chan FileRequest, defFileBufSz)

	c.fileRequests = req
	c.wg = c.startWorkers(ctx, req)
	c.started = true
}

// startWorkers starts download workers.  It returns a sync.WaitGroup.  If the
// req channel is closed, workers will stop, and wg.Wait() completes.
func (c *Client) startWorkers(ctx context.Context, req <-chan FileRequest) *sync.WaitGroup {
	if c.workers == 0 {
		panic("zero workers")
	}
	var wg sync.WaitGroup
	// create workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			c.worker(ctx, fltSeen(req))
			wg.Done()
			dlog.Debugf("download worker %d terminated", workerNum)
		}(i)
	}
	return &wg
}

// worker receives requests from reqC and passes them to saveFile function.
// It will stop if either context is Done, or reqC is closed.
func (c *Client) worker(ctx context.Context, reqC <-chan FileRequest) {
	for {
		select {
		case <-ctx.Done():
			trace.Log(ctx, "warn", "worker context cancelled")
			return
		case req, moar := <-reqC:
			if !moar {
				return
			}
			dlog.Debugf("saving %q to %s, size: %d", filename(req.File), req.Directory, req.File.Size)
			n, err := c.saveFile(ctx, req.Directory, req.File)
			if err != nil {
				dlog.Printf("error saving %q to %s: %s", filename(req.File), req.Directory, err)
				break
			}
			dlog.Printf("file %q saved to %s: %d bytes written", filename(req.File), req.Directory, n)
		}
	}
}

var errNoFS = errors.New("fs adapter not initialised")

// AsyncDownloader starts Client.worker goroutines to download files
// concurrently. It will download any file that is received on fileDlQueue
// channel. It returns the "done" channel and an error. "done" channel will be
// closed once all downloads are complete.
func (c *Client) AsyncDownloader(ctx context.Context, dir string, fileDlQueue <-chan *slack.File) (chan struct{}, error) {
	if c.fs == nil {
		return nil, errNoFS
	}
	done := make(chan struct{})

	req := make(chan FileRequest)
	go func() {
		defer close(req)
		for f := range fileDlQueue {
			req <- FileRequest{Directory: dir, File: f}
		}
	}()

	wg := c.startWorkers(ctx, req)

	// sentinel
	go func() {
		wg.Wait()
		close(done)
	}()

	return done, nil
}

// saveFileWithLimiter saves the file to specified directory, it will use the provided limiter l for throttling.
func (c *Client) saveFile(ctx context.Context, dir string, sf *slack.File) (int64, error) {
	if c.fs == nil {
		return 0, errNoFS
	}
	filePath := filepath.Join(dir, filename(sf))

	var buf bytes.Buffer
	if err := network.WithRetry(ctx, c.limiter, c.retries, func() error {
		region := trace.StartRegion(ctx, "GetFile")
		defer region.End()

		buf.Reset()
		if err := c.client.GetFile(sf.URLPrivateDownload, &buf); err != nil {
			return fmt.Errorf("download to %q failed: %w", filePath, err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	if err := c.fs.WriteFile(filePath, buf.Bytes(), 0666); err != nil {
		return 0, err
	}

	return int64(buf.Len()), nil
}

// filename returns name of the file
func filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	close(c.fileRequests)
	dlog.Debugf("chan closed")
	c.wg.Wait()
	dlog.Debugf("wait complete")

	c.fileRequests = nil
	c.wg = nil
	c.started = false
}

var ErrNotStarted = errors.New("downloader not started")

// DownloadFile requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue, and save the file
// to the directory that was specified when Start was called. If the file buffer
// is full, will block until it becomes empty.
func (c *Client) DownloadFile(dir string, f slack.File) error {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()

	if !started {
		return ErrNotStarted
	}
	c.fileRequests <- FileRequest{Directory: dir, File: &f}
	return nil
}
