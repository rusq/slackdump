package types

import (
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
)

// Message is the internal representation of message with thread.
type Message struct {
	slack.Message
	ThreadReplies []Message `json:"slackdump_thread_replies,omitempty"`
}

func (m Message) Datetime() (time.Time, error) {
	return structures.ParseSlackTS(m.Timestamp)
}

// IsBotMessage returns true if the message is from a bot.
func (m Message) IsBotMessage() bool {
	return m.Msg.BotID != ""
}

func (m Message) IsThread() bool {
	return m.Msg.ThreadTimestamp != ""
}

// IsThreadChild will return true if the message is the parent message of a
// conversation (has more than 0 replies)
func (m Message) IsThreadParent() bool {
	return m.IsThread() && m.Msg.ReplyCount != 0
}

// IsThreadChild will return true if the message is the child message of a
// conversation.
func (m Message) IsThreadChild() bool {
	return m.IsThread() && m.Msg.ReplyCount == 0
}
