package chunk

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/rusq/slackdump/v2/internal/chunk/state"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

var testThreads = []Chunk{
	{
		Type:      CThreadMessages,
		Timestamp: 1234567890,
		ChannelID: "C1234567890",
		IsThread:  true,
		Count:     2,
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
		Type:      CThreadMessages,
		Timestamp: 1234567891,
		ChannelID: "C1234567890",
		IsThread:  true,
		Count:     2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123458",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123458",
					Timestamp:       "1234567890.200000",
					Text:            "Hello, world!",
				},
			},
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123458",
					Timestamp:       "1234567890.300000",
					Text:            "Hello, Slack!",
				},
			},
		},
	},
	{
		Type:      CThreadMessages,
		Timestamp: 1234567890,
		ChannelID: "C1234567890",
		IsThread:  true,
		Count:     2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123456",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.400000",
					Text:            "Hello again world",
				},
			},
			{
				Msg: slack.Msg{
					ThreadTimestamp: "1234567890.123456",
					Timestamp:       "1234567890.500000",
					Text:            "Hello again Slack!",
				},
			},
		},
	},
}

var testThreadsIndex = index{
	"tC1234567890:1234567890.123456": []int64{0, 1209},
	"tC1234567890:1234567890.123458": []int64{604},
}

var testChunks = []Chunk{
	{Type: CChannelInfo, ChannelID: "C1234567890", Channel: &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1234567890"}}}},
	{Type: CMessages, ChannelID: "C1234567890", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.100000", Text: "message1"}},
		{Msg: slack.Msg{Timestamp: "1234567890.200000", Text: "message2"}},
		{Msg: slack.Msg{Timestamp: "1234567890.300000", Text: "message3"}},
		{Msg: slack.Msg{Timestamp: "1234567890.400000", Text: "message4"}},
		{Msg: slack.Msg{Timestamp: "1234567890.500000", Text: "message5"}},
	}},
	{Type: CMessages, ChannelID: "C1234567890", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.600000", Text: "Hello, again!"}},
		{Msg: slack.Msg{Timestamp: "1234567890.700000", Text: "And again!"}},
	}},
	{Type: CMessages, ChannelID: "C1234567890", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.800000", Text: "And again!"}},
		{
			Msg: slack.Msg{
				ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa70",
				ThreadTimestamp: "1234567890.800000",
				Timestamp:       "1234567890.800000",
				Text:            "parent message",
			},
		},
	}},
	{
		Type:      CThreadMessages,
		ChannelID: "C1234567890",
		IsThread:  true,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa70",
				ThreadTimestamp: "1234567890.800000",
				Timestamp:       "1234567890.800000",
				Text:            "parent message",
			},
		},
		Timestamp: 1234567890,
		Count:     2,
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa71",
					Timestamp:       "1234567890.900000",
					ThreadTimestamp: "1234567890.900000",
					Text:            "Hello, world!",
				},
			},
			{
				Msg: slack.Msg{
					ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa72",
					Timestamp:       "1234567891.100000",
					ThreadTimestamp: "1234567890.123456",
					Text:            "Hello, Slack!",
				},
			},
		},
	},
	// chunks from another channel
	{Type: CChannelInfo, ChannelID: "C987654321", Channel: &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C987654321"}}}},
	{Type: CMessages, ChannelID: "C987654321", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.100000", Text: "message1"}},
		{Msg: slack.Msg{Timestamp: "1234567890.200000", Text: "message2"}},
		{Msg: slack.Msg{Timestamp: "1234567890.300000", Text: "message3"}},
		{Msg: slack.Msg{Timestamp: "1234567890.400000", Text: "message4"}},
		{Msg: slack.Msg{Timestamp: "1234567890.500000", Text: "message5"}},
	}},
	{Type: CMessages, ChannelID: "C987654321", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.600000", Text: "Hello, again!"}},
		{Msg: slack.Msg{Timestamp: "1234567890.700000", Text: "And again!"}},
	}},
	{Type: CMessages, ChannelID: "C987654321", Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.800000", Text: "And again!"}},
		{
			Msg: slack.Msg{
				ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa70",
				ThreadTimestamp: "1234567890.800000",
				Timestamp:       "1234567890.800000",
				Text:            "parent message",
			},
		},
	}},
	{
		Type:      CThreadMessages,
		ChannelID: "C987654321",
		IsThread:  true,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa70",
				ThreadTimestamp: "1234567890.800000",
				Timestamp:       "1234567890.800000",
				Text:            "parent message",
			},
		},
		Timestamp: 1234567890,
		Count:     2,
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa71",
					Timestamp:       "1234567890.900000",
					ThreadTimestamp: "1234567890.900000",
					Text:            "Hello, world!",
				},
			},
			{
				Msg: slack.Msg{
					ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa72",
					Timestamp:       "1234567891.100000",
					ThreadTimestamp: "1234567890.123456",
					Text:            "Hello, Slack!",
				},
			},
		},
	},
}

func marshalChunks(t *testing.T, v []Chunk) []byte {
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
				rs: bytes.NewReader(marshalChunks(t, testThreads)),
			},
			want:    testThreadsIndex,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := indexChunks(json.NewDecoder(tt.args.rs))
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
	data := marshalChunks(t, testThreads)
	p := Player{
		rs:      bytes.NewReader(data),
		idx:     testThreadsIndex,
		pointer: make(offsets),
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

func TestPlayer_FileState(t *testing.T) {
	type fields struct {
		rs         io.ReadSeeker
		pointer    offsets
		idx        index
		lastOffset atomic.Int64
	}
	tests := []struct {
		name    string
		fields  fields
		want    *state.State
		wantErr bool
	}{
		{
			name: "single thread",
			fields: fields{
				rs: bytes.NewReader(marshalChunks(t, testThreads)),
			},
			want: &state.State{
				Version:  state.Version,
				Channels: make(map[string]int64),
				Threads: map[string]int64{
					"C1234567890:1234567890.123456": 1234567890500000,
					"C1234567890:1234567890.123458": 1234567890300000,
				},
				Files: make(map[string]string),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Player{
				rs:         tt.fields.rs,
				pointer:    tt.fields.pointer,
				idx:        tt.fields.idx,
				lastOffset: tt.fields.lastOffset,
			}
			got, err := p.State()
			if (err != nil) != tt.wantErr {
				t.Errorf("Player.FileState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("Player.FileState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayer_AllChannels(t *testing.T) {
	type fields struct {
		rs         io.ReadSeeker
		pointer    offsets
		lastOffset atomic.Int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "ok",
			fields: fields{
				rs: bytes.NewReader(marshalChunks(t, testChunks)),
			},
			want: []string{"C1234567890", "C987654321"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := indexChunks(json.NewDecoder(tt.fields.rs))
			if err != nil {
				t.Fatal(err)
			}
			p := &Player{
				rs:         tt.fields.rs,
				idx:        idx,
				pointer:    tt.fields.pointer,
				lastOffset: tt.fields.lastOffset,
			}
			if got := p.AllChannelIDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Player.AllChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testUserChunks = []Chunk{
	{
		Type: CUsers,
		Users: []slack.User{
			{ID: "U1234567890", Name: "user1"},
			{ID: "U987654321", Name: "user2"},
		},
	},
	{
		Type: CUsers,
		Users: []slack.User{
			{ID: "U1234567891", Name: "user3"},
			{ID: "U987654322", Name: "user4"},
		},
	},
	{
		Type: CUsers,
		Users: []slack.User{
			{ID: "U1234567892", Name: "user5"},
			{ID: "U987654323", Name: "user6"},
		},
	},
	{
		Type: CUsers,
		Users: []slack.User{
			{ID: "U1234567893", Name: "user7"},
			{ID: "U987654324", Name: "user8"},
		},
	},
}

func TestPlayer_AllUsers(t *testing.T) {
	type fields struct {
		rs         io.ReadSeeker
		pointer    offsets
		lastOffset atomic.Int64
	}
	tests := []struct {
		name    string
		fields  fields
		want    []slack.User
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				rs: bytes.NewReader(marshalChunks(t, append(testUserChunks, testChunks...))),
			},
			want: []slack.User{
				{ID: "U1234567890", Name: "user1"},
				{ID: "U987654321", Name: "user2"},
				{ID: "U1234567891", Name: "user3"},
				{ID: "U987654322", Name: "user4"},
				{ID: "U1234567892", Name: "user5"},
				{ID: "U987654323", Name: "user6"},
				{ID: "U1234567893", Name: "user7"},
				{ID: "U987654324", Name: "user8"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := indexChunks(json.NewDecoder(tt.fields.rs))
			if err != nil {
				t.Fatal(err)
			}
			p := &Player{
				rs:         tt.fields.rs,
				idx:        idx,
				pointer:    tt.fields.pointer,
				lastOffset: tt.fields.lastOffset,
			}
			got, err := p.AllUsers()
			if (err != nil) != tt.wantErr {
				t.Errorf("Player.AllUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Player.AllUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlayer_offsetTimestamps(t *testing.T) {
	type fields struct {
		rs         io.ReadSeeker
		pointer    offsets
		lastOffset atomic.Int64
	}
	tests := []struct {
		name   string
		fields fields
		want   offts
	}{
		{
			name: "ok",
			fields: fields{
				rs: bytes.NewReader(marshalChunks(t, testChunks)),
			},
			want: offts{
				546:  offsetInfo{ID: "C1234567890", Timestamps: []string{"1234567890.100000", "1234567890.200000", "1234567890.300000", "1234567890.400000", "1234567890.500000"}},
				1382: offsetInfo{ID: "C1234567890", Timestamps: []string{"1234567890.600000", "1234567890.700000"}},
				1751: offsetInfo{ID: "C1234567890", Timestamps: []string{"1234567890.800000", "1234567890.800000"}},
				2208: offsetInfo{ID: "tC1234567890:1234567890.800000", Type: CThreadMessages, Timestamps: []string{"1234567890.900000", "1234567891.100000"}},
				3572: offsetInfo{ID: "C987654321", Timestamps: []string{"1234567890.100000", "1234567890.200000", "1234567890.300000", "1234567890.400000", "1234567890.500000"}},
				4407: offsetInfo{ID: "C987654321", Timestamps: []string{"1234567890.600000", "1234567890.700000"}},
				4775: offsetInfo{ID: "C987654321", Timestamps: []string{"1234567890.800000", "1234567890.800000"}},
				5231: offsetInfo{ID: "tC987654321:1234567890.800000", Type: CThreadMessages, Timestamps: []string{"1234567890.900000", "1234567891.100000"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := indexChunks(json.NewDecoder(tt.fields.rs))
			if err != nil {
				t.Fatal(err)
			}
			p := &Player{
				rs:         tt.fields.rs,
				idx:        idx,
				pointer:    tt.fields.pointer,
				lastOffset: tt.fields.lastOffset,
			}
			got := p.offsetTimestamps()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_timeOffsets(t *testing.T) {
	type args struct {
		ots offts
	}
	tests := []struct {
		name string
		args args
		want map[int64]TimeOffset
	}{
		{
			name: "ok",
			args: args{
				ots: offts{
					546: offsetInfo{ID: "C1234567890", Timestamps: []string{"1234567890.100000", "1234567890.200000", "1234567890.300000", "1234567890.400000", "1234567890.500000"}},
				},
			},
			want: map[int64]TimeOffset{
				1234567890100000: {
					Offset:    546,
					Timestamp: "1234567890.100000",
					Index:     0,
				},
				1234567890200000: {
					Offset:    546,
					Timestamp: "1234567890.200000",
					Index:     1,
				},
				1234567890300000: {
					Offset:    546,
					Timestamp: "1234567890.300000",
					Index:     2,
				},
				1234567890400000: {
					Offset:    546,
					Timestamp: "1234567890.400000",
					Index:     3,
				},
				1234567890500000: {
					Offset:    546,
					Timestamp: "1234567890.500000",
					Index:     4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, timeOffsets(tt.args.ots))
		})
	}
}
