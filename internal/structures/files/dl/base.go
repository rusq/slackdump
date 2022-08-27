package dl

import (
	"context"

	"github.com/rusq/slackdump/v2/logger"
)

const entFiles = "files"

type base struct {
	dl    exportDownloader
	token string // token is the token that will be appended to each file URL.
	l     logger.Interface
}

func (bd *base) Start(ctx context.Context) {
	bd.dl.Start(ctx)
}

func (bd *base) Stop() {
	bd.dl.Stop()
}
