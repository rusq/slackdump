package control

import (
	"context"
	"runtime/trace"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
)

func (s *DirController) SearchMessages(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return s.wrkSearchMessage(ctx, query)
	})
	eg.Go(func() error {
		return s.wrkWorkspace(ctx)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}

func (s *DirController) SearchFiles(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return s.wrkSearchFile(ctx, query)
	})
	eg.Go(func() error {
		return s.wrkWorkspace(ctx)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}

func (s *DirController) SearchAll(ctx context.Context, query string) error {
	var eg errgroup.Group
	start := time.Now()
	eg.Go(func() error {
		return s.wrkSearchMessage(ctx, query)
	})
	eg.Go(func() error {
		return s.wrkSearchFile(ctx, query)
	})
	eg.Go(func() error {
		return s.wrkWorkspace(ctx)
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	s.lg.InfoContext(ctx, "search completed ", "query", query, "took", time.Since(start).String())
	return nil
}

// wrkDirSearchMsg searches messages in the workspace and writes to the directory cd.
func (c *DirController) wrkSearchMessage(ctx context.Context, query string) error {
	ctx, task := trace.NewTask(ctx, "wrkSearchMessage")
	defer task.End()

	search, err := dirproc.NewSearch(c.cd, c.filer)
	if err != nil {
		return err
	}
	defer search.Close()
	return searchMsgWorker(ctx, c.s, search, query)
}

// wrkDirSearchFile searches files in the workspace and writes to the directory cd.
func (c *DirController) wrkSearchFile(ctx context.Context, query string) error {
	ctx, task := trace.NewTask(ctx, "wrkSearchFile")
	defer task.End()

	search, err := dirproc.NewSearch(c.cd, c.filer)
	if err != nil {
		return err
	}
	defer search.Close()

	return searchFileWorker(ctx, c.s, search, query)
}

func (c *DirController) wrkWorkspace(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "wrkWorkspace")
	defer task.End()

	wsproc, err := dirproc.NewWorkspace(c.cd)
	if err != nil {
		return err
	}
	defer wsproc.Close()
	return workspaceWorker(ctx, c.s, wsproc)
}
