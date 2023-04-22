package chunk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

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
				rs: marshalChunks(testThreads...),
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

func TestFile_State(t *testing.T) {
	type fields struct {
		rs  io.ReadSeeker
		idx index
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
				rs: marshalChunks(testThreads...),
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
			p := &File{
				rs:  tt.fields.rs,
				idx: tt.fields.idx,
			}
			got, err := p.State()
			if (err != nil) != tt.wantErr {
				t.Errorf("File.State() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("File.State() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFile_AllChannels(t *testing.T) {
	type fields struct {
		rs io.ReadSeeker
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "ok",
			fields: fields{
				rs: marshalChunks(testChunks...),
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
			p := &File{
				rs:  tt.fields.rs,
				idx: idx,
			}
			if got := p.AllChannelIDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("File.AllChannels() = %v, want %v", got, tt.want)
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

func TestFile_AllUsers(t *testing.T) {
	type fields struct {
		rs io.ReadSeeker
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
				rs: marshalChunks(append(testUserChunks, testChunks...)...),
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
			p := &File{
				rs:  tt.fields.rs,
				idx: idx,
			}
			got, err := p.AllUsers()
			if (err != nil) != tt.wantErr {
				t.Errorf("File.AllUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("File.AllUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFile_offsetTimestamps(t *testing.T) {
	type fields struct {
		rs io.ReadSeeker
	}
	tests := []struct {
		name    string
		fields  fields
		want    offts
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				rs: marshalChunks(testChunks...),
			},
			want: offts{
				546:  offsetInfo{ID: "C1234567890", Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				1382: offsetInfo{ID: "C1234567890", Timestamps: []int64{1234567890600000, 1234567890700000}},
				1751: offsetInfo{ID: "C1234567890", Timestamps: []int64{1234567890800000, 1234567890800000}},
				2208: offsetInfo{ID: "tC1234567890:1234567890.800000", Type: CThreadMessages, Timestamps: []int64{1234567890900000, 1234567891100000}},
				3572: offsetInfo{ID: "C987654321", Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				4407: offsetInfo{ID: "C987654321", Timestamps: []int64{1234567890600000, 1234567890700000}},
				4775: offsetInfo{ID: "C987654321", Timestamps: []int64{1234567890800000, 1234567890800000}},
				5231: offsetInfo{ID: "tC987654321:1234567890.800000", Type: CThreadMessages, Timestamps: []int64{1234567890900000, 1234567891100000}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := indexChunks(json.NewDecoder(tt.fields.rs))
			if err != nil {
				t.Fatal(err)
			}
			p := &File{
				rs:  tt.fields.rs,
				idx: idx,
			}
			got, err := p.offsetTimestamps()
			if (err != nil) != tt.wantErr {
				t.Errorf("File.offsetTimestamps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
		want map[int64]Addr
	}{
		{
			name: "ok",
			args: args{
				ots: offts{
					546: offsetInfo{ID: "C1234567890", Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				},
			},
			want: map[int64]Addr{
				1234567890100000: {
					Offset: 546,
					Index:  0,
				},
				1234567890200000: {
					Offset: 546,
					Index:  1,
				},
				1234567890300000: {
					Offset: 546,
					Index:  2,
				},
				1234567890400000: {
					Offset: 546,
					Index:  3,
				},
				1234567890500000: {
					Offset: 546,
					Index:  4,
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

type sortedArgs struct {
	ts time.Time
	m  *slack.Message
}

func (f sortedArgs) String() string {
	return fmt.Sprintf("fnCallArgs{ts: %v, m: %s}", f.ts, f.m.Text)
}

type sortedArgsSlice []sortedArgs

func (f sortedArgsSlice) String() string {
	var b bytes.Buffer
	for _, v := range f {
		b.WriteString(v.String())
		b.WriteString("\n")
	}
	return b.String()
}

func TestFile_Sorted(t *testing.T) {
	type fields struct {
		rs io.ReadSeeker
	}
	type args struct {
		fn func(ts time.Time, m *slack.Message) error
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantFnCalls sortedArgsSlice
		wantErr     bool
	}{
		{
			name: "ok",
			fields: fields{
				rs: marshalChunks(testChunks...),
			},
			args: args{
				fn: func(ts time.Time, m *slack.Message) error {
					return nil
				},
			},
			wantFnCalls: []sortedArgs{
				{ts: time.Unix(1234567890, 100000000).UTC(), m: &testChunks[1].Messages[0]},
				{ts: time.Unix(1234567890, 200000000).UTC(), m: &testChunks[1].Messages[1]},
				{ts: time.Unix(1234567890, 300000000).UTC(), m: &testChunks[1].Messages[2]},
				{ts: time.Unix(1234567890, 400000000).UTC(), m: &testChunks[1].Messages[3]},
				{ts: time.Unix(1234567890, 500000000).UTC(), m: &testChunks[1].Messages[4]},
				{ts: time.Unix(1234567890, 600000000).UTC(), m: &testChunks[2].Messages[0]},
				{ts: time.Unix(1234567890, 700000000).UTC(), m: &testChunks[2].Messages[1]},
				{ts: time.Unix(1234567890, 800000000).UTC(), m: &testChunks[3].Messages[1]},
				{ts: time.Unix(1234567890, 900000000).UTC(), m: &testChunks[4].Messages[0]},
				{ts: time.Unix(1234567891, 100000000).UTC(), m: &testChunks[4].Messages[1]},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := indexChunks(json.NewDecoder(tt.fields.rs))
			if err != nil {
				t.Fatal(err)
			}
			p := &File{
				rs:  tt.fields.rs,
				idx: idx,
			}
			var rec sortedArgsSlice

			recorder := func(ts time.Time, m *slack.Message) error {
				rec = append(rec, sortedArgs{ts, m})
				return nil
			}

			if err := p.Sorted(context.Background(), false, recorder); (err != nil) != tt.wantErr {
				t.Errorf("File.Sorted() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantFnCalls.String(), rec.String())
		})
	}
}