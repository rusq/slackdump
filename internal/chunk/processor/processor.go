package processor

import (
	"context"
	"io"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

// Conversations is the interface for conversation fetching.
//
//go:generate mockgen -destination ../../mocks/mock_processor/mock_processor.go github.com/rusq/slackdump/v2/internal/chunk/processor Conversations,Users,Channels
type Conversations interface {
	// ChannelInfo is called for each channel that is retrieved.
	ChannelInfo(ctx context.Context, ci *slack.Channel, isThread bool) error
	// Messages is called for each message that is retrieved.
	Messages(ctx context.Context, channelID string, numThreads int, isLast bool, mm []slack.Message) error
	// ThreadMessages is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(ctx context.Context, channelID string, parent slack.Message, isLast bool, tm []slack.Message) error

	Filer
	io.Closer
}

type Filer interface {
	// Files is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(ctx context.Context, channel *slack.Channel, parent slack.Message, isThread bool, ff []slack.File) error
}

var _ Conversations = new(chunk.Recorder)

type Users interface {
	// Users is called for each user chunk that is retrieved.
	Users(ctx context.Context, users []slack.User) error
}

type WorkspaceInfo interface {
	WorkspaceInfo(context.Context, *slack.AuthTestResponse) error
}

var _ Users = new(chunk.Recorder)

type Channels interface {
	// Channels is called for each channel chunk that is retrieved.
	Channels(ctx context.Context, channels []slack.Channel) error
}

var _ Channels = new(chunk.Recorder)

type options struct {
	dumpFiles bool
}

// Option is a functional option for the processor.
type Option func(*options)

// DumpFiles disables the file processing (enabled by default).  It may be
// useful on enterprise workspaces where the file download may be monitored.
// See [#191]
//
// [#191]: https://github.com/rusq/slackdump/discussions/191#discussioncomment-4953235
func DumpFiles(b bool) Option {
	return func(o *options) {
		o.dumpFiles = b
	}
}
