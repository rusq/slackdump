package processors

import (
	"fmt"

	"github.com/slack-go/slack"
)

// EventType is the type of event that was recorded.  There are three types:
// messages, thread messages, and files.
type EventType int

const (
	EventMessages EventType = iota
	EventThreadMessages
	EventFiles
)

// Event is a single event that was recorded.  It contains the type of event,
// the timestamp of the event, the channel ID, and the number of messages or
// files that were recorded.
type Event struct {
	Type            EventType       `json:"type"`
	Timestamp       int64           `json:"event_ts,omitempty"`
	ChannelID       string          `json:"channel_id,omitempty"`
	IsThreadMessage bool            `json:"is_thread_message,omitempty"`
	Size            int             `json:"size,omitempty"` // number of messages or files
	Parent          *slack.Message  `json:"parent,omitempty"`
	Messages        []slack.Message `json:"messages,omitempty"`
	Files           []slack.File    `json:"files,omitempty"`
}

func (e *Event) messageID() string {
	return e.ChannelID
}

func (e *Event) threadID() string {
	return threadID(e.ChannelID, e.Parent.ThreadTimestamp)
}

func threadID(channelID, threadTS string) string {
	return "t" + channelID + ":" + threadTS
}

func (e *Event) fileID() string {
	return fileID(e.ChannelID, e.Parent.Timestamp)
}

func fileID(channelID, parentTS string) string {
	return "f" + channelID + ":" + parentTS
}

// ID returns a unique ID for the event.
func (e *Event) ID() string {
	switch e.Type {
	case EventMessages:
		return e.messageID()
	case EventThreadMessages:
		return e.threadID()
	case EventFiles:
		return e.fileID()
	}
	return fmt.Sprintf("<unknown:%d>", e.Type)
}
