package control

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

func (s *Controller) Search(ctx context.Context, query string) error {
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
