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
	Type            EventType       `json:"_t"`
	Timestamp       int64           `json:"_ts,omitempty"`
	ChannelID       string          `json:"_cid,omitempty"`
	IsThreadMessage bool            `json:"_istm,omitempty"`
	Size            int             `json:"_sz,omitempty"` // number of messages or files
	Parent          *slack.Message  `json:"_p,omitempty"`
	Messages        []slack.Message `json:"_m,omitempty"`
	Files           []slack.File    `json:"_f,omitempty"`
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

// fileEvtID returns a unique ID for the file event.
func (e *Event) fileEvtID() string {
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
		return e.fileEvtID()
	}
	return fmt.Sprintf("<unknown:%d>", e.Type)
}
