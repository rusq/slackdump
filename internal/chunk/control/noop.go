package control

// In this file: dummies for the transformer and filer, if the user chooses
// not to plug in any transformers.

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/processor"
)

type (
	noopFiler            = processor.NopFiler
	noopAvatarProc       = processor.NopAvatars
	noopChannelProcessor = processor.NopChannels
)

type noopExpTransformer struct{}

func (*noopExpTransformer) StartWithUsers(ctx context.Context, users []slack.User) error { return nil }
func (*noopExpTransformer) Transform(context.Context, string, string, bool) error        { return nil }
func (*noopExpTransformer) Wait() error                                                  { return nil }
