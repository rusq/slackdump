package slackdump

import (
	"bufio"
	"fmt"
	"io"
	"time"

	"github.com/slack-go/slack"
)

// minMsgTimeApart defines the time interval in minutes to separate group
// of messages from a single user in the conversation.  This increases the
// readability of the text output.
const minMsgTimeApart = 2

// Channel keeps the slice of messages.
type Channel struct {
	Messages []Message
	ID       string
}

type Message struct {
	slack.Message
	ThreadReplies []Message `json:"slackdump_thread_replies,omitempty"`
}

// ToText outputs Messages m to io.Writer w in text format.
func (m Channel) ToText(sd *SlackDumper, w io.Writer) (err error) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	var (
		prevMsg  Message
		prevTime time.Time
	)
	for _, message := range m.Messages {
		t, err := fromSlackTime(message.Timestamp)
		if err != nil {
			return err
		}
		diff := t.Sub(prevTime)
		if prevMsg.User == message.User && diff.Minutes() < minMsgTimeApart {
			writer.WriteString(fmt.Sprintf(
				"%s\n", message.Text,
			))
		} else {
			writer.WriteString(fmt.Sprintf(
				"\n> %s @ %s:\n%s\n",
				sd.GetUserForMessage(&message),
				t.Format("02/01/2006 15:04:05 Z0700"),
				message.Text,
			))
		}
		prevMsg = message
		prevTime = t

	}
	return nil
}

// GetUserForMessage returns username for the message
func (sd *SlackDumper) GetUserForMessage(msg *Message) string {
	var userid string
	if msg.Comment != nil {
		userid = msg.Comment.User
	} else {
		userid = msg.User
	}

	if userid != "" {
		return sd.username(userid)
	}

	return ""
}
