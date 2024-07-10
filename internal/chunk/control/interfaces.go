package control

import (
	"context"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/processor"
)

// Streamer is the interface for the API scraper.
type Streamer interface {
	Conversations(ctx context.Context, proc processor.Conversations, links <-chan string) error
	ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error
	Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error
	WorkspaceInfo(ctx context.Context, proc processor.WorkspaceInfo) error
	SearchMessages(ctx context.Context, proc processor.MessageSearcher, query string) error
	SearchFiles(ctx context.Context, proc processor.FileSearcher, query string) error
}

type TransformStarter interface {
	StartWithUsers(ctx context.Context, users []slack.User) error
}

// ExportTransformer is a transformer that can be started with a list of
// users.  The compound nature of this interface is called by the asynchronous
// nature of execution and the fact that we need to start the transformer
// after Users goroutine is done, which can happen any time after the Run has
// started.
type ExportTransformer interface {
	dirproc.Transformer
	TransformStarter
}
