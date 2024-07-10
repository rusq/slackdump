package dirproc

import (
	"context"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

// Search is the search results directory processor.  The results are written
// to "search.json.gz" file in the chunk directory.
type Search struct {
	*dirproc

	subproc processor.Filer

	recordFiles bool
}

// NewSearch creates a new search processor.
func NewSearch(dir *chunk.Directory, filer processor.Filer) (*Search, error) {
	p, err := newDirProc(dir, chunk.FSearch)
	if err != nil {
		return nil, err
	}
	return &Search{
		dirproc: p,
		subproc: filer,
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
