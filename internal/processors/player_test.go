package processors

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/slack-go/slack"
)

var testThreads = []Event{
	{
		Type:            EventThreadMessages,
		Timestamp:       1234567890,
		ChannelID:       "C1234567890",
		IsThreadMessage: true,
		Size:            2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123456",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.123456",
					Text:            "Hello, world!",
				},
			},
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.123457",
					Text:            "Hello, Slack!",
				},
			},
		},
	},
	{
		Type:            EventThreadMessages,
		Timestamp:       1234567891,
		ChannelID:       "C1234567890",
		IsThreadMessage: true,
		Size:            2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123458",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123458",
					Timestamp:       "1234567890.123458",
					Text:            "Hello, world!",
				},
			},
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123458",
					Timestamp:       "1234567890.123459",
					Text:            "Hello, Slack!",
				},
			},
		},
	},
	{
		Type:            EventThreadMessages,
		Timestamp:       1234567890,
		ChannelID:       "C1234567890",
		IsThreadMessage: true,
		Size:            2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123456",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.123467",
					Text:            "Hello again world",
				},
			},
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.123468",
					Text:            "Hello again Slack!",
				},
			},
		},
	},
}

var testThreadsIndex = index{
	"tC1234567890:1234567890.123456": []int64{0, 1305},
	"tC1234567890:1234567890.123458": []int64{652},
}

func marshalEvents(t *testing.T, v []Event) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, e := range v {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func Test_indexRecords(t *testing.T) {
	type args struct {
		rs io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    index
		wantErr bool
	}{
		{
			name: "single thread",
			args: args{
				rs: bytes.NewReader(marshalEvents(t, testThreads)),
			},
			want:    testThreadsIndex,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := indexRecords(tt.args.rs)
			if (err != nil) != tt.wantErr {
				t.Errorf("indexRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("indexRecords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayer_Thread(t *testing.T) {
	data := marshalEvents(t, testThreads)
	p := Player{
		rs:      bytes.NewReader(data),
		idx:     testThreadsIndex,
		pointer: make(state),
	}
	m, err := p.Thread("C1234567890", "1234567890.123456")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m))
	}
	// again
	m, err = p.Thread("C1234567890", "1234567890.123456")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(m))
	}
	// should error
	m, err = p.Thread("C1234567890", "1234567890.123456")
	if !errors.Is(err, io.EOF) {
		t.Error(err, "expected io.EOF")
	}
	if len(m) > 0 {
		t.Fatalf("expected 0 messages, got %d", len(m))
	}
}
