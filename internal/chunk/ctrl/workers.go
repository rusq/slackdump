package ctrl

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/chunk/processor/dirproc"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/logger"
)

func userWorker(ctx context.Context, s Streamer, chunkdir *chunk.Directory, tf TransformStarter) error {
	userproc, err := dirproc.NewUsers(chunkdir.Name())
	if err != nil {
		return err
	}
	defer userproc.Close()

	if err := s.Users(ctx, userproc); err != nil {
		return fmt.Errorf("error listing users: %w", err)
	}
	if err := userproc.Close(); err != nil {
		return fmt.Errorf("error closing user processor: %w", err)
	}
	logger.FromContext(ctx).Debug("users done")
	users, err := chunkdir.Users() // load users from chunks
	if err != nil {
		return fmt.Errorf("error loading users: %w", err)
	}
	if err := tf.StartWithUsers(ctx, users); err != nil {
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

func conversationWorker(ctx context.Context, s Streamer, proc processor.Conversations, links <-chan string, resFn ...func(sr slackdump.StreamResult) error) error {
	lg := logger.FromContext(ctx)
	if err := s.Conversations(ctx, proc, links, func(sr slackdump.StreamResult) error {
		for _, fn := range resFn {
			if err := fn(sr); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, transform.ErrClosed) {
			return fmt.Errorf("upstream error: %w", err)
		}
		return fmt.Errorf("error streaming conversations: %w", err)
	}
	lg.Debug("conversations done")
	return nil
}

func workspaceWorker(ctx context.Context, s Streamer, tmpdir string) error {
	lg := logger.FromContext(ctx)
	lg.Debug("workspaceWorker started")
	wsproc, err := dirproc.NewWorkspace(tmpdir)
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
