package event

import (
	"encoding/json"
	"io"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/state"
)

// Recorder is a special Channeler that records all the data it receives, so
// that it can be replayed later.
type Recorder struct {
	w io.Writer

	events chan Event
	errC   chan error

	enc Encoder // encoder to use for the events

	state *state.State
}

// Option is a function that configures the Recorder.
type Option func(r *Recorder)

// WithEncoder allows you to specify a custom encoder to use for the events.
// By default, json.NewEncoder is used.
func WithEncoder(enc Encoder) Option {
	return func(r *Recorder) {
		r.enc = enc
	}
}

func NewRecorder(w io.Writer, options ...Option) *Recorder {
	filename := "unknown"
	if f, ok := w.(namer); ok {
		filename = f.Name()
	}
	rec := &Recorder{
		events: make(chan Event),
		errC:   make(chan error, 1),
		enc:    json.NewEncoder(w),
		state:  state.New(filename),
	}
	for _, opt := range options {
		opt(rec)
	}
	go rec.worker(rec.enc)
	return rec
}

// Encoder is the interface that wraps the Encode method.
type Encoder interface {
	Encode(event interface{}) error
}

func (rec *Recorder) worker(enc Encoder) {
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

// Messages is called for each message chunk that is retrieved.
func (rec *Recorder) Messages(channelID string, m []slack.Message) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.events <- Event{
		Type:      EMessages,
		Timestamp: time.Now().UnixNano(),
		ChannelID: channelID,
		Count:     len(m),
		Messages:  m,
	}: // ok
		for i := range m {
			rec.state.AddMessage(channelID, m[i].Timestamp)
		}
	}
	return nil
}

// Files is called for each file chunk that is retrieved. The parent message is
// passed in as well.
func (rec *Recorder) Files(channelID string, parent slack.Message, isThread bool, f []slack.File) error {
	select {
	case err := <-rec.errC:
		return err
	case rec.events <- Event{
		Type:            EFiles,
		ChannelID:       channelID,
		Parent:          &parent,
		IsThreadMessage: isThread,
		Count:           len(f),
		Files:           f,
	}: // ok
		for i := range f {
			rec.state.AddFile(channelID, f[i].ID)
		}
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
		Type:            EThreadMessages,
		ChannelID:       channelID,
		Parent:          &parent,
		IsThreadMessage: true,
		Count:           len(tm),
		Messages:        tm,
	}: // ok
		for i := range tm {
			rec.state.AddThread(channelID, parent.ThreadTimestamp, tm[i].Timestamp)
		}
	}
	return nil
}

func (rec *Recorder) State() (*state.State, error) {
	return rec.state, nil
}

func (rec *Recorder) Close() error {
	close(rec.events)
	return <-rec.errC
}
