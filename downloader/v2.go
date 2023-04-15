package downloader

import (
	"context"
	"fmt"
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

// ClientV2 is the instance of the downloader.
type ClientV2 struct {
	sc      Downloader
	limiter *rate.Limiter
	fsa     fsadapter.FS
	lg      logger.Interface

	retries int
	workers int

	mu        sync.Mutex // mutex prevents race condition when starting/stopping
	requests  chan request
	chanBufSz int
	wg        *sync.WaitGroup
	started   bool
}

// Option is the function signature for the option functions.
type OptionV2 func(*ClientV2)

// LimiterV2 uses the initialised limiter instead of built in.
func LimiterV2(l *rate.Limiter) OptionV2 {
	return func(c *ClientV2) {
		if l != nil {
			c.limiter = l
		}
	}
}

// RetriesV2 sets the number of attempts that will be taken for the file download.
func RetriesV2(n int) OptionV2 {
	return func(c *ClientV2) {
		if n <= 0 {
			n = defRetries
		}
		c.retries = n
	}
}

// WorkersV2 sets the number of workers for the download queue.
func WorkersV2(n int) OptionV2 {
	return func(c *ClientV2) {
		if n <= 0 {
			n = defNumWorkers
		}
		c.workers = n
	}
}

// Logger allows to use an external log library, that satisfies the
// logger.Interface.
func WithLogger(l logger.Interface) OptionV2 {
	return func(c *ClientV2) {
		if l == nil {
			l = logger.Default
		}
		c.lg = l
	}
}

// NewV2 initialises new file downloader.
func NewV2(sc Downloader, fs fsadapter.FS, opts ...OptionV2) *ClientV2 {
	if sc == nil {
		// better safe than sorry
		panic("programming error:  client is nil")
	}
	c := &ClientV2{
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

type request struct {
	fullpath string
	url      string
}

// Start starts an async file downloader.  If the downloader is already
// started, it does nothing.
func (c *ClientV2) Start(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		// already started
		return
	}
	req := make(chan request, c.chanBufSz)

	c.requests = req
	c.wg = c.startWorkers(ctx, req)
	c.started = true
}

// startWorkers starts download workers.  It returns a sync.WaitGroup.  If the
// req channel is closed, workers will stop, and wg.Wait() completes.
func (c *ClientV2) startWorkers(ctx context.Context, req <-chan request) *sync.WaitGroup {
	if c.workers == 0 {
		c.workers = defNumWorkers
	}
	var wg sync.WaitGroup
	// create workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerNum int) {
			c.worker(ctx, c.fltSeen(req))
			wg.Done()
			c.lg.Debugf("download worker %d terminated", workerNum)
		}(i)
	}
	return &wg
}

// fltSeen filters the files from filesC to ensure that no duplicates
// are downloaded.
func (c *ClientV2) fltSeen(reqC <-chan request) <-chan request {
	filtered := make(chan request)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(filtered)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[string]bool)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for r := range reqC {
			if _, ok := seen[r.url]; ok {
				c.lg.Debugf("already seen %q, skipping", r.url)
				continue
			}
			seen[r.url] = true
			filtered <- r
		}
	}()
	return filtered
}

// worker receives requests from reqC and passes them to saveFile function.
// It will stop if either context is Done, or reqC is closed.
func (c *ClientV2) worker(ctx context.Context, reqC <-chan request) {
	for {
		select {
		case <-ctx.Done():
			trace.Log(ctx, "info", "worker context cancelled")
			return
		case req, moar := <-reqC:
			if !moar {
				return
			}
			c.lg.Debugf("saving %q to %s", path.Base(req.url), req.fullpath)
			n, err := c.download(ctx, req.fullpath, req.url)
			if err != nil {
				c.lg.Printf("error saving %q to %q: %s", path.Base(req.url), req.fullpath, err)
				break
			}
			c.lg.Debugf("file %q saved to %s: %d bytes written", path.Base(req.url), req.fullpath, n)
		}
	}
}

// saveFileWithLimiter saves the file to specified directory, it will use the provided limiter l for throttling.
func (c *ClientV2) download(ctx context.Context, fullpath string, url string) (int64, error) {
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
func (c *ClientV2) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return
	}

	close(c.requests)
	c.lg.Debugf("download files channel closed, waiting for downloads to complete")
	c.wg.Wait()
	c.lg.Debugf("wait complete:  all files downloaded")

	c.requests = nil
	c.wg = nil
	c.started = false
}

// Download requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue.
func (c *ClientV2) Download(fullpath string, url string) error {
	c.mu.Lock()
	started := c.started
	c.mu.Unlock()

	if !started {
		return ErrNotStarted
	}
	c.requests <- request{fullpath: fullpath, url: url}
	return nil
}
