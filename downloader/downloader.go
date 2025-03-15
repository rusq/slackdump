package downloader

import (
	"context"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"log/slog"
	"os"
	"path"
	"runtime/trace"
	"sync"
	"sync/atomic"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v3/internal/network"
)

const (
	defRetries    = 3    // default number of retries if download fails
	defNumWorkers = 4    // number of download processes
	defLimit      = 5000 // default API limit, in events per second.
	defFileBufSz  = 100  // default download channel buffer.
)

var (
	ErrNoFS           = errors.New("fs adapter not initialised")
	ErrNotStarted     = errors.New("downloader not started")
	ErrAlreadyStarted = errors.New("downloader already started")
)

// GetFiler is the file downloader interface.  It exists primarily for mocking
// in tests.
//
//go:generate mockgen -destination=../mocks/mock_downloader/mock_getfiler.go . GetFiler
type GetFiler interface {
	// GetFile retrieves a given file from its private download URL
	GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error
}

// Client is the instance of the downloader.
type Client struct {
	sc  GetFiler
	fsa fsadapter.FS

	requests chan Request
	done     chan struct{} // when all workers complete, this channel gets a message.

	mu      sync.Mutex // mutex prevents race condition when starting/stopping
	started atomic.Bool

	options
}

// Option is the function signature for the option functions.
type Option func(*options)

type options struct {
	limiter   *rate.Limiter
	retries   int
	workers   int
	lg        *slog.Logger
	chanBufSz int
}

// FilenameFunc is the file naming function that should return the output
// filename for slack.File.
type FilenameFunc func(*slack.File) string

// Filename returns name of the file generated from the slack.File.
var Filename FilenameFunc = stdFilenameFn

// Limiter uses the initialised limiter instead of built in.
func Limiter(l *rate.Limiter) Option {
	return func(c *options) {
		if l != nil {
			c.limiter = l
		}
	}
}

// Retries sets the number of attempts that will be taken for the file download.
func Retries(n int) Option {
	return func(c *options) {
		if n <= 0 {
			n = defRetries
		}
		c.retries = n
	}
}

// Workers sets the number of workers for the download queue.
func Workers(n int) Option {
	return func(c *options) {
		if n <= 0 {
			n = defNumWorkers
		}
		c.workers = n
	}
}

// Logger allows to use an external log library, that satisfies the
// *slog.Logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *options) {
		if l == nil {
			l = slog.Default()
		}
		c.lg = l
	}
}

// New initialises new file downloader.
func New(sc GetFiler, fs fsadapter.FS, opts ...Option) *Client {
	if sc == nil {
		// better safe than sorry
		panic("programming error:  client is nil")
	}
	c := &Client{
		sc:   sc,
		fsa:  fs,
		done: make(chan struct{}, 1),
		options: options{
			lg:        slog.Default(),
			limiter:   rate.NewLimiter(defLimit, 1),
			retries:   defRetries,
			workers:   defNumWorkers,
			chanBufSz: defFileBufSz,
		},
	}
	for _, opt := range opts {
		opt(&c.options)
	}
	return c
}

type Request struct {
	Fullpath string
	URL      string
}

// Start starts an async file downloader.  If the downloader is already
// started, it does nothing.
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started.Load() {
		return ErrAlreadyStarted
	}
	c.lg.Debug("starting downloader")
	c.startWorkers(ctx)
	c.started.Store(true)
	return nil
}

// startWorkers starts download workers.  It returns a sync.WaitGroup.  If the
// req channel is closed, workers will stop, and wg.Wait() completes.
func (c *Client) startWorkers(ctx context.Context) {
	if c.workers == 0 {
		c.workers = defNumWorkers
	}
	c.requests = make(chan Request, defFileBufSz)
	var wg sync.WaitGroup
	seen := fltSeen(c.requests, 0)
	// create workers
	for i := range c.workers {
		wg.Add(1)
		slog.DebugContext(ctx, "started worker", "i", i)
		go func(workerNum int) {
			c.worker(ctx, seen)
			wg.Done()
			c.lg.DebugContext(ctx, "download worker terminated", "worker", workerNum)
		}(i)
	}
	go func() {
		// start sentinel
		wg.Wait()
		c.done <- struct{}{}
	}()
}

// fltSeen filters the files from filesC to ensure that no duplicates
// are downloaded.
func fltSeen(reqC <-chan Request, bufSz int) <-chan Request {
	filtered := make(chan Request)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(filtered)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[uint64]struct{}, bufSz)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for r := range reqC {
			h := hash(r.URL + r.Fullpath)
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			filtered <- r
		}
		slog.Debug("all files processed")
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
	// we deliberately not handling context cancellation here, because we want
	// to drain the reqC channel, so that Download would unlock the mutex
	// and not wait to send on the request channel that no worker is servicing
	// due to exiting by context cancellation.
	for req := range reqC {
		lg := c.lg.With("filename", path.Base(req.URL), "destination", req.Fullpath)
		lg.DebugContext(ctx, "saving file")
		n, err := c.download(ctx, req.Fullpath, req.URL)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				lg.DebugContext(ctx, "download cancelled")
			} else {
				lg.ErrorContext(ctx, "error saving file", "error", err)
			}
		} else {
			lg.DebugContext(ctx, "file saved", "bytes_written", n)
		}
	}
	slog.DebugContext(ctx, "worker exiting")
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

	if err := network.WithRetry(ctx, c.limiter, c.retries, func(ctx context.Context) error {
		region := trace.StartRegion(ctx, "GetFile")
		defer region.End()

		if err := c.sc.GetFileContext(ctx, url, tf); err != nil {
			if _, err := tf.Seek(0, io.SeekStart); err != nil {
				c.lg.ErrorContext(ctx, "seek", "error", err)
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

	if !c.started.CompareAndSwap(true, false) {
		return
	}

	slog.Debug("mutex locked, stopping downloader")
	close(c.requests)

	c.lg.Debug("requests channel closed, waiting for all downloads to complete")
	<-c.done
	c.lg.Debug("wait complete:  no more files to download")

	c.requests = nil
}

// Download requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue.
func (c *Client) Download(fullpath string, url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started.Load() {
		return ErrNotStarted
	}

	c.requests <- Request{Fullpath: fullpath, URL: url}

	return nil
}

func (c *Client) AsyncDownloader(ctx context.Context, queueC <-chan Request) (<-chan struct{}, error) {
	done := make(chan struct{})
	if err := c.Start(ctx); err != nil {
		close(done)
		return done, err
	}
	go func() {
		defer close(done)
		for r := range queueC {
			if err := c.Download(r.Fullpath, r.URL); err != nil {
				c.lg.Error("download error", "url", r.URL, "error", err)
			}
		}
		c.Stop()
	}()

	return done, nil
}

func stdFilenameFn(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}
