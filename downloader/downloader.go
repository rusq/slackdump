package downloader

import (
	"context"
	"fmt"
	"hash/crc64"
	"io"
	"os"
	"path"
	"runtime/trace"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/logger"
	"golang.org/x/time/rate"
)

// Client is the instance of the downloader.
type Client struct {
	sc      Downloader
	limiter *rate.Limiter
	fsa     fsadapter.FS
	lg      logger.Interface

	retries int
	workers int

	mu        sync.Mutex // mutex prevents race condition when starting/stopping
	requests  chan Request
	chanBufSz int
	wg        *sync.WaitGroup
	started   bool
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
func WithLogger(l logger.Interface) Option {
	return func(c *Client) {
		if l == nil {
			l = logger.Default
		}
		c.lg = l
	}
}

// New initialises new file downloader.
func New(sc Downloader, fs fsadapter.FS, opts ...Option) *Client {
	if sc == nil {
		// better safe than sorry
		panic("programming error:  client is nil")
	}
	c := &Client{
		sc:        sc,
		fsa:       fs,
		limiter:   rate.NewLimiter(defLimit, 1),
		lg:        logger.Default,
		chanBufSz: defFileBufSz,
		retries:   defRetries,
		workers:   defNumWorkers,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Request struct {
	Fullpath string
	URL      string
}

// Start starts an async file downloader.  If the downloader is already
// started, it does nothing.
func (c *Client) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		// already started
		return
	}
	req := make(chan Request, c.chanBufSz)

	c.requests = req
	c.wg = c.startWorkers(ctx, req)
	c.started = true
}

// startWorkers starts download workers.  It returns a sync.WaitGroup.  If the
// req channel is closed, workers will stop, and wg.Wait() completes.
func (c *Client) startWorkers(ctx context.Context, req <-chan Request) *sync.WaitGroup {
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
			c.lg.Debugf("download worker %d terminated", workerNum)
		}(i)
	}
	return &wg
}

// fltSeen filters the files from filesC to ensure that no duplicates
// are downloaded.
func fltSeen(reqC <-chan Request) <-chan Request {
	filtered := make(chan Request)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(filtered)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[uint64]bool, 1000)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for r := range reqC {
			h := hash(r.URL + r.Fullpath)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = true
			filtered <- r
		}
	}()
	return filtered
}

var crctab = crc64.MakeTable(crc64.ISO)

func hash(s string) uint64 {
	h := crc64.New(crctab)
	h.Write([]byte(s))
	return h.Sum64()
}

// worker receives requests from reqC and passes them to saveFile function.
// It will stop if either context is Done, or reqC is closed.
func (c *Client) worker(ctx context.Context, reqC <-chan Request) {
	for {
		select {
		case <-ctx.Done():
			trace.Log(ctx, "info", "worker context cancelled")
			return
		case req, moar := <-reqC:
			if !moar {
				return
			}
			c.lg.Debugf("saving %q to %s", path.Base(req.URL), req.Fullpath)
			n, err := c.download(ctx, req.Fullpath, req.URL)
			if err != nil {
				c.lg.Printf("error saving %q to %q: %s", path.Base(req.URL), req.Fullpath, err)
				break
			}
			c.lg.Debugf("file %q saved to %s: %d bytes written", path.Base(req.URL), req.Fullpath, n)
		}
	}
}

// saveFileWithLimiter saves the file to specified directory, it will use the provided limiter l for throttling.
func (c *Client) download(ctx context.Context, fullpath string, url string) (int64, error) {
	if c.fsa == nil {
		return 0, ErrNoFS
	}

	tf, err := os.CreateTemp("", "")
	if err != nil {
		return 0, err
	}
	defer func() {
		tf.Close()
		os.Remove(tf.Name())
	}()

	if err := network.WithRetry(ctx, c.limiter, c.retries, func() error {
		region := trace.StartRegion(ctx, "GetFile")
		defer region.End()

		if err := c.sc.GetFile(url, tf); err != nil {
			if _, err := tf.Seek(0, io.SeekStart); err != nil {
				c.lg.Debugf("seek error: %s", err)
			}
			return fmt.Errorf("download to %q failed, [src=%s]: %w", fullpath, url, err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// at this point, temporary file position would be at EOF, we need to reset
	// it prior to copying.
	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	fsf, err := c.fsa.Create(fullpath)
	if err != nil {
		return 0, err
	}
	defer fsf.Close()

	n, err := io.Copy(fsf, tf)
	if err != nil {
		return 0, err
	}

	return int64(n), nil
}

// Stop waits for all transfers to finish, and stops the downloader.
func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	close(c.requests)
	c.lg.Debugf("requests channel closed, waiting for all downloads to complete")
	c.wg.Wait()
	c.lg.Debugf("wait complete:  all files downloaded")

	c.requests = nil
	c.wg = nil
	c.started = false
}

// Download requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue.
func (c *Client) Download(fullpath string, url string) error {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()

	if !started {
		return ErrNotStarted
	}
	c.requests <- Request{Fullpath: fullpath, URL: url}
	return nil
}

func (c *Client) AsyncDownloader(ctx context.Context, queueC <-chan Request) (<-chan struct{}, error) {
	done := make(chan struct{})
	c.Start(ctx)
	go func() {
		defer close(done)
		for r := range queueC {
			if err := c.Download(r.Fullpath, r.URL); err != nil {
				c.lg.Printf("error downloading %q: %s", r.URL, err)
			}
		}
	}()

	return done, nil
}
