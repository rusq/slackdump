package processor

import (
	"io"

	"github.com/slack-go/slack"
)

// Processor is the interface for conversation fetching.
type Processor interface {
	// Messages is called for each message that is retrieved.
	Messages(channelID string, m []slack.Message) error
	// Files is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(channelID string, parent slack.Message, isThread bool, m []slack.File) error
	// ThreadMessages is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(channelID string, parent slack.Message, tm []slack.Message) error

	io.Closer
}
