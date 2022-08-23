package export

import (
	"context"
	"net/url"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/logger"
)

const entFiles = "files"

//go:generate sh -c "mockgen -source files.go -destination files_mock_test.go -package export"

// fileProcessor is the file exporter interface.
type fileProcessor interface {
	// ProcessFunc returns the process function that should be passed to
	// DumpMessagesRaw. It should be able to extract files from the messages
	// and download them.  If the downloader is not started, i.e. if file
	// download is disabled, it should silently ignore the error and return
	// nil.
	ProcessFunc(channelName string) slackdump.ProcessFunc
}

type fileExporter interface {
	fileProcessor
	startStopper
}

type startStopper interface {
	Start(ctx context.Context)
	Stop()
}

// exportDownloader is the interface that downloader.Client implements.  Used
// for mocking in tests.
type exportDownloader interface {
	DownloadFile(dir string, f slack.File) (string, error)
	startStopper
}

type baseDownloader struct {
	dl    exportDownloader
	token string // token is the token that will be appended to each file URL.
	l     logger.Interface
}

func (bd *baseDownloader) Start(ctx context.Context) {
	bd.dl.Start(ctx)
}

func (bd *baseDownloader) Stop() {
	bd.dl.Stop()
}

func newFileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface, token string) fileExporter {
	switch t {
	case TNoDownload:
		return newFileUpdater(token)
	default:
		l.Printf("unknown export type %s, using standard format", t)
		fallthrough
	case TStandard:
		return newStdDl(fs, cl, l, token)
	case TMattermost:
		return newMattermostDl(fs, cl, l, token)
	}
}

// addToken updates the uri, adding the t= query parameter with token value.
// if token or url is empty, it does nothing.
func addToken(uri string, token string) (string, error) {
	if token == "" || uri == "" {
		return uri, nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	val := u.Query()
	val.Set("t", token)
	u.RawQuery = val.Encode()
	return u.String(), nil
}

// updateTokenFn returns a file update function that adds the t= query parameter
// with token value. If token value is empty, the function does nothing.
func updateTokenFn(token string) func(*slack.File) error {
	return func(f *slack.File) error {
		if token == "" {
			return nil
		}
		var err error
		update := func(s *string, t string) {
			if err != nil {
				return
			}
			*s, err = addToken(*s, t)
		}
		update(&f.URLPrivate, token)
		update(&f.URLPrivateDownload, token)
		update(&f.Thumb64, token)
		update(&f.Thumb80, token)
		update(&f.Thumb160, token)
		update(&f.Thumb360, token)
		update(&f.Thumb360Gif, token)
		update(&f.Thumb480, token)
		update(&f.Thumb720, token)
		update(&f.Thumb960, token)
		update(&f.Thumb1024, token)
		return nil
	}
}
