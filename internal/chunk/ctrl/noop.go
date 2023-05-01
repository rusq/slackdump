package ctrl

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

type noopFiler struct{}

func (n *noopFiler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	return nil
}

type noopTransformer struct{}

func (n *noopTransformer) StartWithUsers(ctx context.Context, users []slack.User) error {
	return nil
}

func (n *noopTransformer) Transform(ctx context.Context, id chunk.FileID) error {
	return nil
}
