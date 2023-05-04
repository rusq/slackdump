package control

import (
	"context"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v2/processor"
	"github.com/slack-go/slack"
)

type Streamer interface {
	Conversations(ctx context.Context, proc processor.Conversations, links <-chan string, fn func(slackdump.StreamResult) error) error
	ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error
	Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error
	WorkspaceInfo(ctx context.Context, proc processor.WorkspaceInfo) error
}

type TransformStarter interface {
	dirproc.Transformer
	StartWithUsers(ctx context.Context, users []slack.User) error
}
