package downloader

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/internal/network"
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
		panic("programming error:  client is nil")
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

// SaveFileTo saves a single file to the specified directory.
func (c *Client) SaveFileTo(ctx context.Context, dir string, f *slack.File) (int64, error) {
	return c.saveFile(ctx, dir, f)
}

// AsyncDownloader starts Client.worker goroutines to download files
// concurrently. It will download any file that is received on fileDlQueue
// channel. It returns the "done" channel and an error. "done" channel will
// be closed once all downloads are complete.
func (c *Client) AsyncDownloader(ctx context.Context, dir string, fileDlQueue <-chan *slack.File) (chan struct{}, error) {
	done := make(chan struct{})

	if dir == "" {
		return nil, errors.New("empty directory")
	}

	if err := os.Mkdir(dir, 0777); err != nil {
		if !os.IsExist(err) {
			close(done)
			return done, err
		}
	}

	var wg sync.WaitGroup
	// create workers
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			c.worker(ctx, dir, seenFilter(fileDlQueue))
			wg.Done()
		}()
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
			return err
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
			dlog.Debugf("saving %s, size: %d", filename(file), file.Size)
			n, err := c.saveFile(ctx, dir, file)
			if err != nil {
				dlog.Printf("error saving %q: %s", filename(file), err)
				break
			}
			dlog.Printf("file %s saved: %d bytes written", filename(file), n)
		}
	}
}

// seenFilter filters the files from filesC to ensure that no duplicates
// are downloaded.
func seenFilter(filesC <-chan *slack.File) <-chan *slack.File {
	dlQ := make(chan *slack.File)
	go func() {
		// closing stop will lead to all worker goroutines to terminate.
		defer close(dlQ)

		// seen contains file ids that already been seen,
		// so we don't download the same file twice
		seen := make(map[string]bool)
		// files queue must be closed by the caller (see DumpToDir.(1))
		for f := range filesC {
			if _, ok := seen[f.ID]; ok {
				log.Printf("already seen %s, skipping", filename(f))
				continue
			}
			seen[f.ID] = true
			dlQ <- f
		}
	}()
	return dlQ
}
