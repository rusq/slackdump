package control

// In this file: dummies for the transformer and filer, if the user chooses
// not to plug in any transformers.

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type noopFiler struct{}

func (n *noopFiler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	return nil
}
func (n *noopFiler) Close() error { return nil }

type noopTransformer struct{}

func (n *noopTransformer) StartWithUsers(ctx context.Context, users []slack.User) error { return nil }
func (n *noopTransformer) Transform(ctx context.Context, id chunk.FileID) error         { return nil }
func (n *noopTransformer) Wait() error                                                  { return nil }

type noopAvatarProc struct{}

func (n *noopAvatarProc) Users(ctx context.Context, users []slack.User) error { return nil }
func (n *noopAvatarProc) Close() error                                        { return nil }
