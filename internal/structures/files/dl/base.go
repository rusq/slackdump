package dl

import (
	"context"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/structures/files"
	"github.com/rusq/slackdump/v2/logger"
)

const entFiles = "files"

// exportDownloader is the interface that dl.Client implements.  Used
// for mocking in tests.
type exportDownloader interface {
	DownloadFile(dir string, f slack.File) (string, error)
	files.StartStopper
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
