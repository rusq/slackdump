package control

import (
	"context"
	"time"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/logger"
	"golang.org/x/sync/errgroup"
)

type Search struct {
	cd *chunk.Directory
	s  *slackdump.Stream
	lg logger.Interface
}

func NewSearch(cd *chunk.Directory, s *slackdump.Stream) *Search {
	return &Search{cd: cd, s: s, lg: logger.Default}
}

func (s *Search) Search(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return searchWorker(ctx, s.s, s.cd, query)
	})
	eg.Go(func() error {
		return workspaceWorker(ctx, s.s, s.cd)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.Printf("search for query %q completed in: %s", query, time.Since(start))
	return nil
}
