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
	errC   chan error
}

func NewRecorder(w io.Writer) *Recorder {
	rec := &Recorder{
		w:      w,
		events: make(chan Event),
		errC:   make(chan error, 1),
	}
	go rec.worker(json.NewEncoder(rec.w))
	return rec
}

type encoder interface {
	Encode(v interface{}) error
}

func (rec *Recorder) worker(enc encoder) {
LOOP:
	for event := range rec.events {
		if err := enc.Encode(event); err != nil {
			select {
			case rec.errC <- err:
			default:
				// unable to send, prevent deadlock
				break LOOP
			}
		}
	}
	close(rec.errC)
}

// Messages is called for each message that is retrieved.
func (rec *Recorder) Messages(channelID string, m []slack.Message) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.events <- Event{
		Type:      EventMessages,
		Timestamp: time.Now().UnixNano(),
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
	case err := <-rec.errC:
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
	case err := <-rec.errC:
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
	return <-rec.errC
}
