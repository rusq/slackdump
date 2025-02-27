package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/processor"
)

// Special processors for the controller.

// userCollector collects users and sends the signal to start the transformer.
type userCollector struct {
	ctx   context.Context // bad boy, but short-lived, so it's ok
	users []slack.User
	ts    TransformStarter
}

var _ processor.Users = (*userCollector)(nil)

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

// conversationTransformer monitors the "last" messages, and once
// the reference count drops to zero, it starts the transformer for
// the channel.  It should be appended after the main conversation
// processor.
type conversationTransformer struct {
	ctx context.Context
	tf  dirproc.Transformer
	rc  ReferenceChecker
}

var _ processor.Messenger = (*conversationTransformer)(nil)

func (ct *conversationTransformer) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error {
	if isLast {
		return ct.mbeTransform(ctx, channelID, "", false)
	}
	return nil
}

func (ct *conversationTransformer) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly bool, isLast bool, tm []slack.Message) error {
	if isLast {
		return ct.mbeTransform(ctx, channelID, parent.ThreadTimestamp, threadOnly)
	}
	return nil
}

func (ct *conversationTransformer) mbeTransform(ctx context.Context, channelID, threadID string, threadOnly bool) error {
	finalised, err := ct.rc.IsFinalised(ctx, channelID)
	if err != nil {
		return fmt.Errorf("error checking if finalised: %w", err)
	}
	if !finalised {
		return nil
	}
	if err := ct.tf.Transform(ctx, chunk.ToFileID(channelID, threadID, threadOnly)); err != nil {
		return fmt.Errorf("error transforming: %w", err)
	}
	return nil
}
