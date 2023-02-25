package processor

import (
	"context"
	"io"

	"github.com/slack-go/slack"
)

// Processor is the interface for conversation fetching.
type Processor interface {
	// Messages is called for each message that is retrieved.
	Messages(ctx context.Context, channelID string, m []slack.Message) error
	// Files is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(ctx context.Context, channelID string, parent slack.Message, isThread bool, m []slack.File) error
	// ThreadMessages is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(ctx context.Context, channelID string, parent slack.Message, tm []slack.Message) error

	io.Closer
}

type options struct {
	dumpFiles bool
}

// Option is a functional option for the processor.
type Option func(*options)

// DumpFiles disables the file processing (enabled by default).  It may be
// useful on enterprise workspaces where the file download may be monitored.
// See:
// https://github.com/rusq/slackdump/discussions/191#discussioncomment-4953235
func DumpFiles(b bool) Option {
	return func(o *options) {
		o.dumpFiles = b
	}
}
