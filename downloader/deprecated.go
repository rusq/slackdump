// Package downloader provides the sync and async file download functionality.
package downloader

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"

	"errors"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	defRetries    = 3    // default number of retries if download fails
	defNumWorkers = 4    // number of download processes
	defLimit      = 5000 // default API limit, in events per second.
	defFileBufSz  = 100  // default download channel buffer.
)

// ClientV1 is the instance of the downloader.
//
// Deprecated: Use Client.
type ClientV1 struct {
	v2     *Client
	nameFn FilenameFunc
}

// FilenameFunc is the file naming function that should return the output
// filename for slack.File.
type FilenameFunc func(*slack.File) string

// Filename returns name of the file generated from the slack.File.
var Filename FilenameFunc = stdFilenameFn

// Downloader is the file downloader interface.  It exists primarily for mocking
// in tests.
type Downloader interface {
	// GetFile retreives a given file from its private download URL
	GetFile(downloadURL string, writer io.Writer) error
}

// OptionV1 is the function signature for the option functions.
type OptionV1 func(*ClientV1)

// LimiterV1 uses the initialised limiter instead of built in.
func LimiterV1(l *rate.Limiter) OptionV1 {
	return func(c *ClientV1) {
		Limiter(l)(c.v2)
	}
}

// RetriesV1 sets the number of attempts that will be taken for the file download.
func RetriesV1(n int) OptionV1 {
	return func(c *ClientV1) {
		Retries(n)(c.v2)
	}
}

// WorkersV1 sets the number of workers for the download queue.
func WorkersV1(n int) OptionV1 {
	return func(c *ClientV1) {
		Workers(n)(c.v2)
	}
}

// LoggerV1 allows to use an external log library, that satisfies the
// logger.Interface.
func LoggerV1(l logger.Interface) OptionV1 {
	return func(c *ClientV1) {
		WithLogger(l)(c.v2)
	}
}

func WithNameFunc(fn FilenameFunc) OptionV1 {
	return func(c *ClientV1) {
		if fn != nil {
			c.nameFn = fn
		} else {
			c.nameFn = Filename
		}
	}
}

// NewV1 initialises new file downloader.
//
// Deprecated: use NewV2 instead.
func NewV1(client Downloader, fs fsadapter.FS, opts ...OptionV1) *ClientV1 {
	c := &ClientV1{
		v2:     New(client, fs),
		nameFn: Filename,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SaveFile saves a single file to the specified directory synchrounously.
func (c *ClientV1) SaveFile(ctx context.Context, dir string, f *slack.File) (int64, error) {
	return c.v2.download(ctx, filepath.Join(dir, c.nameFn(f)), f.URLPrivateDownload)
}

// Start starts an async file downloader.  If the downloader is already
// started, it does nothing.
func (c *ClientV1) Start(ctx context.Context) {
	c.v2.Start(ctx)
}

var ErrNoFS = errors.New("fs adapter not initialised")

// AsyncDownloader starts Client.worker goroutines to download files
// concurrently. It will download any file that is received on fileDlQueue
// channel. It returns the "done" channel and an error. "done" channel will be
// closed once all downloads are complete.
func (c *ClientV1) AsyncDownloader(ctx context.Context, dir string, fileDlQueue <-chan *slack.File) (<-chan struct{}, error) {
	if c.v2.fsa == nil {
		return nil, ErrNoFS
	}
	dlq := make(chan Request, c.v2.chanBufSz)
	go func() {
		defer close(dlq)
		for f := range fileDlQueue {
			dlq <- Request{
				Fullpath: path.Join(dir, c.nameFn(f)),
				URL:      f.URLPrivateDownload,
			}
		}
	}()
	done, err := c.v2.AsyncDownloader(ctx, dlq)
	if err != nil {
		return nil, err
	}

	return done, nil
}

func stdFilenameFn(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

var ErrNotStarted = errors.New("downloader not started")

// DownloadFile requires a started downloader, otherwise it will return
// ErrNotStarted. Will place the file to the download queue, and save the file
// to the directory that was specified when Start was called. If the file buffer
// is full, will block until it becomes empty.  It returns the filepath within the
// filesystem.
func (c *ClientV1) DownloadFile(dir string, f slack.File) (string, error) {
	path := filepath.Join(dir, c.nameFn(&f))
	if err := c.v2.Download(path, f.URLPrivateDownload); err != nil {
		return "", err
	}
	return path, nil
}

func (c *ClientV1) Stop() {
	c.v2.Stop()
}
