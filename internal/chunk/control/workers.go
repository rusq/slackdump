package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"

	"github.com/rusq/slackdump/v3/internal/structures"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/processor"
)

func userWorker(ctx context.Context, s Streamer, chunkdir *chunk.Directory, tf TransformStarter) error {
	var users = make([]slack.User, 0, 100)
	userproc, err := dirproc.NewUsers(chunkdir, dirproc.WithUsers(func(us []slack.User) error {
		users = append(users, us...)
		return nil
	}))
	if err != nil {
		return err
	}
	if err := s.Users(ctx, userproc); err != nil {
		if err2 := userproc.Close(); err2 != nil {
			err = errors.Join(err2)
		}
		return fmt.Errorf("error listing users: %w", err)
	}
	if err := userproc.Close(); err != nil {
		return fmt.Errorf("error closing user processor: %w", err)
	}
	slog.DebugContext(ctx, "users done")
	if len(users) == 0 {
		return fmt.Errorf("unable to proceed, no users found")
	}
	if err := tf.StartWithUsers(ctx, users); err != nil {
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

func conversationWorker(ctx context.Context, s Streamer, proc processor.Conversations, links <-chan structures.EntityItem) error {
	lg := slog.Default()
	if err := s.Conversations(ctx, proc, links); err != nil {
		if errors.Is(err, transform.ErrClosed) {
			return fmt.Errorf("upstream error: %w", err)
		}
		return fmt.Errorf("error streaming conversations: %w", err)
	}
	lg.Debug("conversations done")
	return nil
}

func workspaceWorker(ctx context.Context, s Streamer, cd *chunk.Directory) error {
	lg := slog.Default()
	lg.Debug("workspaceWorker started")
	wsproc, err := dirproc.NewWorkspace(cd)
	if err != nil {
		return err
	}
	defer wsproc.Close()
	if err := s.WorkspaceInfo(ctx, wsproc); err != nil {
		return err
	}
	lg.Debug("workspaceWorker done")
	return nil
}

func searchMsgWorker(ctx context.Context, s Streamer, filer processor.Filer, cd *chunk.Directory, query string) error {
	ctx, task := trace.NewTask(ctx, "searchMsgWorker")
	defer task.End()

	lg := slog.Default()
	lg.Debug("searchMsgWorker started")
	search, err := dirproc.NewSearch(cd, filer)
	if err != nil {
		return err
	}
	defer search.Close()
	if err := s.SearchMessages(ctx, search, query); err != nil {
		return err
	}
	lg.Debug("searchWorker done")
	return nil
}

func searchFileWorker(ctx context.Context, s Streamer, filer processor.Filer, cd *chunk.Directory, query string) error {
	ctx, task := trace.NewTask(ctx, "searchMsgWorker")
	defer task.End()

	lg := slog.Default()
	lg.Debug("searchFileWorker started")
	search, err := dirproc.NewSearch(cd, filer)
	if err != nil {
		return err
	}
	defer search.Close()
	if err := s.SearchFiles(ctx, search, query); err != nil {
		return err
	}
	lg.Debug("searchFileWorker done")
	return nil
}
