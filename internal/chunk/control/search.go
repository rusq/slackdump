package control

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
)

func (s *Controller) SearchMessages(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return searchMsgWorker(ctx, s.s, s.filer, s.cd, query)
	})
	eg.Go(func() error {
		wsproc, err := dirproc.NewWorkspace(s.cd)
		if err != nil {
			return err
		}
		defer wsproc.Close()
		return workspaceWorker(ctx, s.s, wsproc)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}

func (s *Controller) SearchFiles(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return searchFileWorker(ctx, s.s, s.filer, s.cd, query)
	})
	eg.Go(func() error {
		wsproc, err := dirproc.NewWorkspace(s.cd)
		if err != nil {
			return Error{"workspace", "init", err}
		}
		defer wsproc.Close()
		return workspaceWorker(ctx, s.s, wsproc)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}

func (s *Controller) SearchAll(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return searchMsgWorker(ctx, s.s, s.filer, s.cd, query)
	})
	eg.Go(func() error {
		return searchFileWorker(ctx, s.s, s.filer, s.cd, query)
	})
	eg.Go(func() error {
		wsproc, err := dirproc.NewWorkspace(s.cd)
		if err != nil {
			return Error{"workspace", "init", err}
		}
		defer wsproc.Close()
		return workspaceWorker(ctx, s.s, wsproc)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}
