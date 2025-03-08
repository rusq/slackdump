package control

// In this file: dummies for the transformer and filer, if the user chooses
// not to plug in any transformers.

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/processor"
)

type (
	noopFiler            = processor.NopFiler
	noopAvatarProc       = processor.NopAvatars
	noopChannelProcessor = processor.NopChannels
)

type noopTransformer struct{}

func (n *noopTransformer) StartWithUsers(ctx context.Context, users []slack.User) error { return nil }
func (n *noopTransformer) Transform(ctx context.Context, id chunk.FileID) error         { return nil }
func (n *noopTransformer) Wait() error                                                  { return nil }
