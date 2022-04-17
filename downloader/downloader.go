package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const (
	defDownloadDir = "." // default download directory is current.
	defRetries     = 3   // default number of retries if download fails
	defNumWorkers  = 4   // number of download processes
	defLimit       = 5000
)

type Client struct {
	client  Downloader
	limiter *rate.Limiter

	retries int
	workers int

	mu           *sync.Mutex
	filesDlQueue chan *slack.File
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
func New(client Downloader, opts ...Option) *Client {
	if client == nil {
		// better safe than sorry
		panic("programming error: client is nil")
	}
	c := &Client{
		client:  client,
		limiter: rate.NewLimiter(5000, 1),
		retries: defRetries,
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

// AsyncDownloader starts Client.worker goroutines to download files
// concurrently. It will download any file that is received on fileDlQueue
// channel. It returns the "done" channel and an error. "done" channel will
// be closed once all downloads are complete.
func (c *Client) AsyncDownloader(ctx context.Context, dir string, fileDlQueue <-chan *slack.File) (chan struct{}, error) {
	done := make(chan struct{})

	wg, err := c.start(ctx, dir, fileDlQueue)
	if err != nil {
		close(done)
		return done, nil
	}

	// sentinel
	go func() {
		wg.Wait()
		close(done)
	}()

	return done, nil
}

// saveFileWithLimiter saves the file to specified directory, it will use the provided limiter l for throttling.
func (c *Client) saveFile(ctx context.Context, dir string, sf *slack.File) (int64, error) {
	filePath := filepath.Join(dir, filename(sf))
	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	if err := network.WithRetry(ctx, c.limiter, c.retries, func() error {
		region := trace.StartRegion(ctx, "GetFile")
		defer region.End()

		if err := c.client.GetFile(sf.URLPrivateDownload, f); err != nil {
			// cleanup if download failed.
			f.Close()
			if e := os.RemoveAll(filePath); e != nil {
				trace.Logf(ctx, "error", "removing file after unsuccesful download failed with: %s", e)
			}
			return fmt.Errorf("download to %q failed: %w", filePath, err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return int64(sf.Size), nil
}

// filename returns name of the file
func filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

func (c *Client) worker(ctx context.Context, dir string, filesC <-chan *slack.File) {
	for {
		select {
		case <-ctx.Done():
			trace.Log(ctx, "warn", "worker context cancelled")
			return
		case file, moar := <-filesC:
			if !moar {
				return
			}
			dlog.Debugf("saving %q to %s, size: %d", filename(file), dir, file.Size)
			n, err := c.saveFile(ctx, dir, file)
			if err != nil {
				dlog.Printf("error saving %q to %s: %s", filename(file), dir, err)
				break
			}
			dlog.Printf("file %q saved to %s: %d bytes written", filename(file), dir, n)
		}
	}
}

// Start starts an async file downloader.  If the downloader
// is already started, it does nothing.
func (c *Client) Start(ctx context.Context, dir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		// already started
		return nil
	}
	filesDlQueue := make(chan *slack.File, 20)

	wg, err := c.start(ctx, dir, filesDlQueue)
	if err != nil {
		return err
	}

	c.filesDlQueue = filesDlQueue
	c.wg = wg
	c.started = true
	return nil
}

// start creates a download directory if it does not exist and starts download
// workers.  It returns a sync.WaitGroup.  If the filesDlQueue channel is
// closed, workers will stop, and wg.Wait() completes.
func (c *Client) start(ctx context.Context, dir string, filesDlQueue <-chan *slack.File) (*sync.WaitGroup, error) {
	if err := c.mkdir(dir); err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	// create workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			c.worker(ctx, dir, seenFilter(filesDlQueue))
			wg.Done()
		}()
	}

	return &wg, nil
}

func (c *Client) mkdir(name string) error {
	if name == "" {
		return errors.New("empty directory")
	}

	if err := os.Mkdir(name, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.filesDlQueue)
	c.wg.Wait()

	c.filesDlQueue = nil
	c.wg = nil
	c.started = false
}

var ErrNotStarted = errors.New("downloader not started")

// DownloadFile requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue, and save the file
// to the directory that was specified when Start was called. If the file buffer
// is full, will block until it becomes empty.
func (c *Client) DownloadFile(f *slack.File) error {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()

	if !started {
		return ErrNotStarted
	}
	c.filesDlQueue <- f
	return nil
}
