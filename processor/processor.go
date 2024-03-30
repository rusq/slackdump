package processor

import (
	"context"
	"io"

	"github.com/rusq/slack"
)

// Conversations is the interface for conversation fetching with files.
//
//go:generate mockgen -destination ../mocks/mock_processor/mock_processor.go github.com/rusq/slackdump/v3/processor Conversations,Users,Channels,ChannelInformer,Filer
type Conversations interface {
	Messenger
	Filer
	ChannelInformer

	io.Closer
}

type ChannelInformer interface {
	// ChannelInfo is called for each channel that is retrieved.  ChannelInfo
	// will be called for each direct thread link, and in this case, threadID
	// will be set to the parent message's timestamp.
	ChannelInfo(ctx context.Context, ci *slack.Channel, threadID string) error
	ChannelUsers(ctx context.Context, channelID string, threadTS string, users []string) error
}

// Messenger is the interface that implements only the message fetching.
type Messenger interface {
	// Messages method is called for each message that is retrieved.
	Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error
	// ThreadMessages method is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error
}

type Filer interface {
	// Files method is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error
}

type Users interface {
	// Users method is called for each user chunk that is retrieved.
	Users(ctx context.Context, users []slack.User) error
}

type WorkspaceInfo interface {
	WorkspaceInfo(context.Context, *slack.AuthTestResponse) error
}

type Channels interface {
	// Channels is called for each channel chunk that is retrieved.
	Channels(ctx context.Context, channels []slack.Channel) error
}
