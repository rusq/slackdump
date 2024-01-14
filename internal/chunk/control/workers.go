package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/processor"
	"github.com/slack-go/slack"
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
	logger.FromContext(ctx).Debug("users done")
	if len(users) == 0 {
		return fmt.Errorf("unable to proceed, no users found")
	}
	if err := tf.StartWithUsers(ctx, users); err != nil {
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

func conversationWorker(ctx context.Context, s Streamer, proc processor.Conversations, links <-chan string) error {
	lg := logger.FromContext(ctx)
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
	lg := logger.FromContext(ctx)
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
