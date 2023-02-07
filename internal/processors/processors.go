package processors

import (
	"io"

	"github.com/slack-go/slack"
)

// Channeler is the interface for conversation fetching.
type Channeler interface {
	// Messages is called for each message that is retrieved.
	Messages(m []slack.Message) error
	// Files is called for each file that is retrieved. The parent message is
	// passed in as well.
	Files(parent slack.Message, isThread bool, m []slack.File) error
	// ThreadMessages is called for each of the thread messages that are
	// retrieved. The parent message is passed in as well.
	ThreadMessages(parent slack.Message, tm []slack.Message) error

	io.Closer
}
