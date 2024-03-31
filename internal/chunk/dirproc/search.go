package dirproc

import (
	"context"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

type Search struct {
	*baseproc

	subproc processor.Filer

	recordFiles bool
}

func NewSearch(dir *chunk.Directory, filer processor.Filer) (*Search, error) {
	p, err := newBaseProc(dir, "search")
	if err != nil {
		return nil, err
	}
	return &Search{
		baseproc: p,
		subproc:  filer,
	}, nil
}

func (s *Search) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	if err := s.subproc.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	if !s.recordFiles {
		return nil
	}
	if err := s.Files(ctx, channel, parent, ff); err != nil {
		return err
	}
	return nil
}
