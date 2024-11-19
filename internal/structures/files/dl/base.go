package dl

import (
	"context"
	"log/slog"
)

const entFiles = "files"

type base struct {
	dl    exportDownloader
	token string // token is the token that will be appended to each file URL.
	l     *slog.Logger
}

func (bd *base) Start(ctx context.Context) {
	bd.dl.Start(ctx)
}

func (bd *base) Stop() {
	bd.dl.Stop()
}
