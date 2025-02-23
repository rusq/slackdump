package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v3/internal/structures"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/processor"
)

type userCollector struct {
	ctx   context.Context // bad boy, but short-lived, so it's ok
	users []slack.User
	ts    TransformStarter
}

func (u *userCollector) Users(ctx context.Context, users []slack.User) error {
	u.users = append(u.users, users...)
	return nil
}

func (u *userCollector) Close() error {
	if len(u.users) == 0 {
		return errors.New("no users collected")
	}
	if err := u.ts.StartWithUsers(u.ctx, u.users); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

func userWorker(ctx context.Context, s Streamer, up processor.Users) error {
	if err := s.Users(ctx, up); err != nil {
		return fmt.Errorf("error listing users: %w", err)
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

func workspaceWorker(ctx context.Context, s Streamer, wsproc processor.WorkspaceInfo) error {
	lg := slog.Default()
	lg.Debug("workspaceWorker started")

	if err := s.WorkspaceInfo(ctx, wsproc); err != nil {
		return err
	}
	lg.Debug("workspaceWorker done")
	return nil
}

func searchMsgWorker(ctx context.Context, s Streamer, ms processor.MessageSearcher, query string) error {
	lg := slog.Default()
	lg.Debug("searchMsgWorker started")
	if err := s.SearchMessages(ctx, ms, query); err != nil {
		return err
	}
	lg.Debug("searchWorker done")
	return nil
}

func searchFileWorker(ctx context.Context, s Streamer, sf processor.FileSearcher, query string) error {
	lg := slog.Default()
	lg.Debug("searchFileWorker started")
	if err := s.SearchFiles(ctx, sf, query); err != nil {
		return err
	}
	lg.Debug("searchFileWorker done")
	return nil
}
