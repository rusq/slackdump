package control

import (
	"context"
	"io"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/processor"
)

//go:generate mockgen -destination=mock_control/mock_interfaces.go . Streamer,TransformStarter,ExportTransformer,ReferenceChecker,EncodeReferenceCloser

// Streamer is the interface for the API scraper.
type Streamer interface {
	Conversations(ctx context.Context, proc processor.Conversations, links <-chan structures.EntityItem) error
	ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error
	Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error
	WorkspaceInfo(ctx context.Context, proc processor.WorkspaceInfo) error
	SearchMessages(ctx context.Context, proc processor.MessageSearcher, query string) error
	SearchFiles(ctx context.Context, proc processor.FileSearcher, query string) error
	UsersBulk(ctx context.Context, proc processor.Users, ids ...string) error
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
	chunk.Transformer
	TransformStarter
}

// ReferenceChecker is an interface that contains functions to check if all
// messages for the channel were processed.
type ReferenceChecker interface {
	// IsComplete should return true, if all messages and threads for the
	// channel has been processed.
	IsComplete(ctx context.Context, channelID string) (bool, error)
	// IsCompleteThread should return true, if all messages in the thread
	// for thread-only list entry have been processed.  The behaviour of
	// this function is undefined for non-thread-only list entries.
	IsCompleteThread(ctx context.Context, channelID string, threadID string) (bool, error)
}

// EncodeReferenceCloser is an interface that combines the chunk.Encoder,
// ReferenceChecker and io.Closer interfaces.
type EncodeReferenceCloser interface {
	chunk.Encoder
	ReferenceChecker
	io.Closer
}
