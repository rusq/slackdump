package transform

import (
	"context"
	"errors"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/source"
)

type HTML struct {
	src source.Sourcer
	fsa fsadapter.FS
}

func NewHTML(src source.Sourcer, fsa fsadapter.FS) *HTML {
	return &HTML{
		src: src,
		fsa: fsa,
	}
}

func (h *HTML) Convert(ctx context.Context, id chunk.FileID) error {
	return errors.New("not implemented")
}
