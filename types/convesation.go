package types

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
)

// time format for text output.
const textTimeFmt = "02/01/2006 15:04:05 Z0700"

const (
	// minMsgTimeApart defines the time interval in minutes to separate group
	// of messages from a single user in the conversation.  This increases the
	// readability of the text output.
	minMsgTimeApart = 2 * time.Minute
)

// Conversation keeps the slice of messages.
type Conversation struct {
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
	// ID is the channel ID.
	ID string `json:"channel_id"`
	// ThreadTS is a thread timestamp.  If it's not empty, it means that it's a
	// dump of a thread, not a channel.
	ThreadTS string `json:"thread_ts,omitempty"`
}

func (c Conversation) String() string {
	if c.ThreadTS == "" {
		return c.ID
	}
	return c.ID + "-" + c.ThreadTS
}

func (c Conversation) IsThread() bool {
	return c.ThreadTS != ""
}

// ToText outputs Messages m to io.Writer w in text format.
func (c Conversation) ToText(w io.Writer, userIdx structures.UserIndex) (err error) {
	buf := bufio.NewWriter(w)
	defer buf.Flush()

	return generateText(w, c.Messages, "", userIdx)
}

func generateText(w io.Writer, m []Message, prefix string, userIdx structures.UserIndex) error {
	var (
		prevMsg  Message
		prevTime time.Time
	)
	for _, message := range m {
		t, err := structures.ParseSlackTS(message.Timestamp)
		if err != nil {
			return err
		}
		diff := t.Sub(prevTime)
		if prevMsg.User == message.User && diff < minMsgTimeApart {
			fmt.Fprintf(w, prefix+"%s\n", message.Text)
		} else {
			fmt.Fprintf(w, prefix+"\n"+prefix+"> %s [%s] @ %s:\n%s\n",
				userIdx.Sender(&message.Message), message.User,
				t.Format(textTimeFmt),
				prefix+html.UnescapeString(message.Text),
			)
		}
		if len(message.ThreadReplies) > 0 {
			if err := generateText(w, message.ThreadReplies, "|   ", userIdx); err != nil {
				return err
			}
		}
		prevMsg = message
		prevTime = t
	}
	return nil
}

// UserIDs returns a slice of user IDs.
func (c Conversation) UserIDs() []string {
	var seen = make(map[string]bool, len(c.Messages))
	for _, m := range c.Messages {
		if seen[m.User] {
			continue
		}
		seen[m.User] = true
	}
	return toslice(seen)
}
