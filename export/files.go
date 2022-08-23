package export

import (
	"context"

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

type baseDownloader struct {
	dl exportDownloader
	l  logger.Interface
}

// exportDownloader is the interface that downloader.Client implements.  Used
// for mocking in tests.
type exportDownloader interface {
	DownloadFile(dir string, f slack.File) (string, error)
	startStopper
}

func (bd *baseDownloader) Start(ctx context.Context) {
	bd.dl.Start(ctx)
}

func (bd *baseDownloader) Stop() {
	bd.dl.Stop()
}

func newFileExporter(t ExportType, fs fsadapter.FS, cl *slack.Client, l logger.Interface) fileExporter {
	switch t {
	default:
		l.Printf("unknown export type %s, using standard format", t)
		fallthrough
	case TStandard:
		return newStdDl(fs, cl, l)
	case TMattermost:
		return newMattermostDl(fs, cl, l)
	}
}
