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

type EventType int

const (
	EventMessages EventType = iota
	EventThreadMessages
	EventFiles
)

type Event struct {
	Type            EventType       `json:"type,omitempty"`
	TS              int64           `json:"event_ts,omitempty"`
	IsThreadMessage bool            `json:"is_thread_message,omitempty"`
	Size            int             `json:"size,omitempty"` // number of messages or files
	Parent          *slack.Message  `json:"parent,omitempty"`
	Messages        []slack.Message `json:"messages,omitempty"`
	Files           []slack.File    `json:"files,omitempty"`
}

func NewRecorder(wc io.Writer) *Recorder {
	rec := &Recorder{
		w:      wc,
		events: make(chan Event),
		resp:   make(chan error, 1),
	}
	go rec.worker()
	return rec
}

func (rec *Recorder) worker() {
	enc := json.NewEncoder(rec.w)
	enc.SetIndent("", "  ")
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
func (rec *Recorder) Messages(m []slack.Message) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:     EventMessages,
		TS:       time.Now().UnixNano(),
		Size:     len(m),
		Messages: m}: // ok
	}
	return nil
}

// Files is called for each file that is retrieved. The parent message is
// passed in as well.
func (rec *Recorder) Files(parent slack.Message, isThread bool, f []slack.File) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:            EventFiles,
		Parent:          &parent,
		IsThreadMessage: isThread,
		Size:            len(f),
		Files:           f}: // ok
	}
	return nil
}

// ThreadMessages is called for each of the thread messages that are
// retrieved. The parent message is passed in as well.
func (rec *Recorder) ThreadMessages(parent slack.Message, tm []slack.Message) error {
	select {
	case err := <-rec.resp:
		return err
	case rec.events <- Event{
		Type:            EventThreadMessages,
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
