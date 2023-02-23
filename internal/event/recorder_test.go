package event

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/internal/state"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestEvent_ID(t *testing.T) {
	type fields struct {
		Type            EventType
		TS              int64
		ChannelID       string
		IsThreadMessage bool
		Size            int
		Parent          *slack.Message
		Messages        []slack.Message
		Files           []slack.File
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Message",
			fields{
				Type:      EMessages,
				ChannelID: "C123",
			},
			"C123",
		},
		{
			"Thread",
			fields{
				Type:      EThreadMessages,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{ThreadTimestamp: "123.456"},
				},
			},
			"tC123:123.456",
		},
		{
			"File",
			fields{
				Type:      EFiles,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{Timestamp: "123.456"},
				},
			},
			"fC123:123.456",
		},
		{
			"Unknown type",
			fields{
				Type:      EventType(1000),
				ChannelID: "C123",
			},
			"<unknown:1000>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{
				Type:            tt.fields.Type,
				Timestamp:       tt.fields.TS,
				ChannelID:       tt.fields.ChannelID,
				IsThreadMessage: tt.fields.IsThreadMessage,
				Count:           tt.fields.Size,
				Parent:          tt.fields.Parent,
				Messages:        tt.fields.Messages,
				Files:           tt.fields.Files,
			}
			if got := e.ID(); got != tt.want {
				t.Errorf("Event.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errEncoder struct {
	err error
}

func (e *errEncoder) Encode(v interface{}) error {
	return e.err
}

func TestRecorder_worker(t *testing.T) {
	t.Parallel()
	t.Run("no events", func(t *testing.T) {
		t.Parallel()
		r := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
		}
		time.AfterFunc(40*time.Millisecond, func() {
			close(r.events)
		})
		var buf bytes.Buffer // we don't really need it.
		start := time.Now()
		r.worker(json.NewEncoder(&buf))
		if time.Since(start) > 50*time.Millisecond {
			t.Errorf("worker took too long to exit")
		}
	})
	t.Run("one event", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
			state:  state.New(""),
			w:      &buf,
		}
		go func() {
			r.events <- Event{
				Type:      EMessages,
				ChannelID: "C123",
				Messages:  []slack.Message{{Msg: slack.Msg{Text: "hello"}}},
			}
			close(r.events)
		}()
		start := time.Now()
		r.worker(json.NewEncoder(r.w))
		if time.Since(start) > 50*time.Millisecond {
			t.Errorf("worker took too long to exit")
		}
		const want = `{"_t":0,"_ts":0,"_id":"C123","_c":0,"_m":[{"text":"hello","replace_original":false,"delete_original":false,"metadata":{"event_type":"","event_payload":null},"blocks":null}]}
`
		if !assert.Equal(t, want, buf.String()) {
			t.Errorf("unexpected output: %s", buf.String())
		}
	})
	t.Run("one event, error", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
			w:      &buf,
		}
		go func() {
			r.events <- Event{
				Type:      EMessages,
				ChannelID: "C123",
				Messages:  []slack.Message{{Msg: slack.Msg{Text: "hello"}}},
			}
			close(r.events)
		}()

		start := time.Now()
		r.worker(&errEncoder{err: errors.New("test error")})

		if time.Since(start) > 50*time.Millisecond {
			t.Errorf("worker took too long to exit")
		}
		gotErr := <-r.errC
		if gotErr == nil {
			t.Errorf("expected error, got none")
		}
	})
	t.Run("unsendable error", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error), // unbuffered, and we don't read it.
			w:      &buf,
		}
		go func() {
			r.events <- Event{
				Type:      EMessages,
				ChannelID: "C123",
				Messages:  []slack.Message{{Msg: slack.Msg{Text: "hello"}}},
			}
			close(r.events)
		}()

		r.worker(&errEncoder{err: errors.New("test error")})

		var gotErr error
		time.AfterFunc(1*time.Second, func() { gotErr = <-r.errC }) // give it time to brew the error.
		if gotErr != nil {
			t.Errorf("expected nothing, got error: %v", gotErr)
		}
	})
}

func TestRecorder_Messages(t *testing.T) {
	t.Parallel()
	t.Run("sending a message", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
			state:  state.New(""), // we don't really need it.
		}
		if err := rec.Messages("C123", []slack.Message{{Msg: slack.Msg{Text: "hello"}}}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		evt := <-rec.events
		if evt.Type != EMessages {
			t.Errorf("unexpected event type: %v", evt.Type)
		}
		if evt.ChannelID != "C123" {
			t.Errorf("unexpected channel ID: %s", evt.ChannelID)
		}
		if len(evt.Messages) != 1 {
			t.Errorf("unexpected number of messages: %d", len(evt.Messages))
		}
		if evt.Messages[0].Text != "hello" {
			t.Errorf("unexpected message text: %s", evt.Messages[0].Text)
		}
	})
	t.Run("sending a message, error", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event),
			errC:   make(chan error, 1),
			state:  state.New(""), // we don't really need it.
		}
		rec.errC <- errors.New("test error")
		gotErr := rec.Messages("C123", []slack.Message{{Msg: slack.Msg{Text: "hello"}}})
		if gotErr == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestRecorder_ThreadMessages(t *testing.T) {
	t.Parallel()
	t.Run("sending a message", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
			state:  state.New(""), // we don't really need it.
		}
		if err := rec.ThreadMessages(
			"C123",
			slack.Message{Msg: slack.Msg{Text: "parent"}},
			[]slack.Message{{Msg: slack.Msg{Text: "hello"}}},
		); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		evt := <-rec.events
		if evt.Type != EThreadMessages {
			t.Errorf("unexpected event type: %v", evt.Type)
		}
		if evt.ChannelID != "C123" {
			t.Errorf("unexpected channel ID: %s", evt.ChannelID)
		}
		if evt.Parent.Text != "parent" {
			t.Errorf("unexpected parent text: %s", evt.Parent.Text)
		}
		if len(evt.Messages) != 1 {
			t.Errorf("unexpected number of messages: %d", len(evt.Messages))
		}
		if evt.Messages[0].Text != "hello" {
			t.Errorf("unexpected message text: %s", evt.Messages[0].Text)
		}
	})
	t.Run("sending a message, error", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event),
			errC:   make(chan error, 1),
		}
		rec.errC <- errors.New("test error")
		gotErr := rec.ThreadMessages("C123", slack.Message{Msg: slack.Msg{Text: "parent"}}, []slack.Message{{Msg: slack.Msg{Text: "hello"}}})
		if gotErr == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestRecorder_Files(t *testing.T) {
	t.Parallel()
	t.Run("sending a message", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event, 1),
			errC:   make(chan error, 1),
			state:  state.New(""), // we don't really need it.
		}
		if err := rec.Files(
			"C123",
			slack.Message{Msg: slack.Msg{Text: "parent"}},
			true,
			[]slack.File{{ID: "F123", Name: "file.txt"}},
		); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		evt := <-rec.events
		if evt.Type != EFiles {
			t.Errorf("unexpected event type: %v", evt.Type)
		}
		if evt.ChannelID != "C123" {
			t.Errorf("unexpected channel ID: %s", evt.ChannelID)
		}
		if evt.Parent.Text != "parent" {
			t.Errorf("unexpected parent text: %s", evt.Parent.Text)
		}
		if len(evt.Files) != 1 {
			t.Errorf("unexpected number of messages: %d", len(evt.Messages))
		}
		if evt.Files[0].ID != "F123" {
			t.Errorf("unexpected message text: %s", evt.Messages[0].Text)
		}
	})
	t.Run("sending a message, error", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event),
			errC:   make(chan error, 1),
		}
		rec.errC <- errors.New("test error")
		gotErr := rec.Files(
			"C123",
			slack.Message{Msg: slack.Msg{Text: "parent"}},
			true,
			[]slack.File{{ID: "F123", Name: "file.txt"}},
		)
		if gotErr == nil {
			t.Errorf("expected error, got none")
		}
	})
}

func TestRecorder_Close(t *testing.T) {
	t.Parallel()
	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event),
			errC:   make(chan error, 1),
		}
		time.AfterFunc(10*time.Millisecond, func() {
			close(rec.errC)
		})
		if err := rec.Close(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		rec := &Recorder{
			events: make(chan Event),
			errC:   make(chan error, 1),
		}
		rec.errC <- errors.New("test error")
		if err := rec.Close(); err == nil {
			t.Errorf("expected error, got none")
		}
	})
}
