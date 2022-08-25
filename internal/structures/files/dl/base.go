package dl

import (
	"context"

	"github.com/rusq/slackdump/v2/logger"
)

const entFiles = "files"

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
