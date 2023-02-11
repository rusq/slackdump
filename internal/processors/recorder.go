package processors

import (
	"encoding/json"
	"io"
	"time"

	"github.com/slack-go/slack"
)

// Recorder is a special Channeler that records all the data it receives, so
// that it can be replayed later.
type Recorder struct {
	w io.Writer

	events chan Event
	resp   chan error
}

// EventType is the type of event that was recorded.  There are three types:
// messages, thread messages, and files.
type EventType int

const (
	EventMessages EventType = iota
	EventThreadMessages
	EventFiles
)

type Event struct {
	Type            EventType       `json:"type,omitempty"`
	TS              int64           `json:"event_ts,omitempty"`
	ChannelID       string          `json:"channel_id,omitempty"`
	IsThreadMessage bool            `json:"is_thread_message,omitempty"`
	Size            int             `json:"size,omitempty"` // number of messages or files
	Parent          *slack.Message  `json:"parent,omitempty"`
	Messages        []slack.Message `json:"messages,omitempty"`
	Files           []slack.File    `json:"files,omitempty"`
}

func threadID(channelID string, threadTS string) string {
	return "t" + channelID + ":" + threadTS
}

func (e *Event) ID() string {
	switch e.Type {
	case EventMessages:
		return e.ChannelID
	case EventThreadMessages:
		return threadID(e.ChannelID, e.Parent.Timestamp)
	case EventFiles:
		return "f" + e.ChannelID + ":" + e.Parent.Timestamp
	}
	return "<empty>"
}

func NewRecorder(w io.Writer) *Recorder {
	rec := &Recorder{
		w:      w,
		events: make(chan Event),
		resp:   make(chan error, 1),
	}
	go rec.worker()
	return rec
}

func (rec *Recorder) worker() {
	enc := json.NewEncoder(rec.w)
LOOP:
	for event := range rec.events {
		if err := enc.Encode(event); err != nil {
			select {
			case rec.resp <- err:
			default:
				// unable to send, prevent deadlock
				break LOOP
			}
		}
	}
	close(rec.resp)
}

// Messages is called for each message that is retrieved.
func (rec *Recorder) Messages(channelID string, m []slack.Message) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:      EventMessages,
		TS:        time.Now().UnixNano(),
		ChannelID: channelID,
		Size:      len(m),
		Messages:  m}: // ok
	}
	return nil
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (rec *Recorder) Files(channelID string, parent slack.Message, isThread bool, f []slack.File) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:            EventFiles,
		ChannelID:       channelID,
		Parent:          &parent,
		IsThreadMessage: isThread,
		Size:            len(f),
		Files:           f}: // ok
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (rec *Recorder) ThreadMessages(channelID string, parent slack.Message, tm []slack.Message) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:            EventThreadMessages,
		ChannelID:       channelID,
		Parent:          &parent,
		IsThreadMessage: true,
		Size:            len(tm),
		Messages:        tm}: // ok
	}
	return nil
}

func (rec *Recorder) Close() error {
	close(rec.events)
	return <-rec.resp
}
