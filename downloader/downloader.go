// Package downloader provides the sync and async file download functionality.
package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	defDownloadDir = "."  // default download directory is current.
	defRetries     = 3    // default number of retries if download fails
	defNumWorkers  = 4    // number of download processes
	defLimit       = 5000 // default API limit, in events per second.
	defFileBufSz   = 100  // default download channel buffer.
)

// Client is the instance of the downloader.
type Client struct {
	client  Downloader
	limiter *rate.Limiter
	fs      fsadapter.FS
	dlog    logger.Interface

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

// Option is the function signature for the option functions.
type Option func(*Client)

// Limiter uses the initialised limiter instead of built in.
func Limiter(l *rate.Limiter) Option {
	return func(c *Client) {
		if l != nil {
			c.limiter = l
		}
	}
}

// Retries sets the number of attempts that will be taken for the file download.
func Retries(n int) Option {
	return func(c *Client) {
		if n <= 0 {
			n = defRetries
		}
		c.retries = n
	}
}

// Workers sets the number of workers for the download queue.
func Workers(n int) Option {
	return func(c *Client) {
		if n <= 0 {
			n = defNumWorkers
		}
		c.workers = n
	}
}

// Logger allows to use an external log library, that satisfies the
// logger.Interface.
func Logger(l logger.Interface) Option {
	return func(c *Client) {
		if l == nil {
			l = logger.Default
		}
		c.dlog = l
	}
}

// New initialises new file downloader.
func New(client Downloader, fs fsadapter.FS, opts ...Option) *Client {
	if client == nil {
		// better safe than sorry
		panic("programming error:  client is nil")
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
		c.workers = defNumWorkers
	}
	var wg sync.WaitGroup
	// create workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			c.worker(ctx, fltSeen(req))
			wg.Done()
			c.l().Debugf("download worker %d terminated", workerNum)
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
			trace.Log(ctx, "info", "worker context cancelled")
			return
		case req, moar := <-reqC:
			if !moar {
				return
			}
			c.l().Debugf("saving %q to %s, size: %d", Filename(req.File), req.Directory, req.File.Size)
			n, err := c.saveFile(ctx, req.Directory, req.File)
			if err != nil {
				c.l().Printf("error saving %q to %s: %s", Filename(req.File), req.Directory, err)
				break
			}
			c.l().Printf("file %q saved to %s: %d bytes written", Filename(req.File), req.Directory, n)
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
	filePath := filepath.Join(dir, Filename(sf))

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

// Filename returns name of the file
func Filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

// Stop stops the downloader.
func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	close(c.fileRequests)
	c.l().Debugf("chan closed")
	c.wg.Wait()
	c.l().Debugf("wait complete")

	c.fileRequests = nil
	c.wg = nil
	c.started = false
}

var ErrNotStarted = errors.New("downloader not started")

// DownloadFile requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue, and save the file
// to the directory that was specified when Start was called. If the file buffer
// is full, will block until it becomes empty.  It returns the filepath within the
// filesystem.
func (c *Client) DownloadFile(dir string, f slack.File) (string, error) {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()

	if !started {
		return "", ErrNotStarted
	}
	c.fileRequests <- FileRequest{Directory: dir, File: &f}
	return path.Join(dir, Filename(&f)), nil
}

func (c *Client) l() logger.Interface {
	if c.dlog == nil {
		return logger.Default
	}
	return c.dlog
}
