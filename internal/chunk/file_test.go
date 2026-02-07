// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package chunk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

const (
	TestChannelID  = "C1234567890"
	TestChannelID2 = "C987654321"
)

var testThreads = []Chunk{
	{
		Type:      CThreadMessages,
		Timestamp: 1234567890,
		ChannelID: TestChannelID,
		ThreadTS:  "1234567890.123456",
		Count:     2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123456",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					Timestamp:       "1234567890.123456",
					ThreadTimestamp: "1234567890.123456",
				},
			},
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
		ChannelID: TestChannelID,
		ThreadTS:  "1234567890.123458",
		Count:     2,
		Parent: &slack.Message{
			Msg: slack.Msg{
				ThreadTimestamp: "1234567890.123458",
			},
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					Timestamp:       "1234567890.123458",
					ThreadTimestamp: "1234567890.123458",
				},
			},
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
		ChannelID: TestChannelID,
		ThreadTS:  "1234567890.123456",
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
				},
			},
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
	"tC1234567890:1234567890.123456": []int64{0, 1583},
	"tC1234567890:1234567890.123458": []int64{791},
}

var archivedChannel = []Chunk{
	{Type: CChannelInfo, ChannelID: TestChannelID, Channel: &slack.Channel{GroupConversation: slack.GroupConversation{IsArchived: true, Conversation: slack.Conversation{ID: TestChannelID}}}},
	{Type: CMessages, ChannelID: TestChannelID, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.100000", Text: "message1"}},
		{Msg: slack.Msg{Timestamp: "1234567890.200000", Text: "message2"}},
		{Msg: slack.Msg{Timestamp: "1234567890.300000", Text: "message3"}},
		{Msg: slack.Msg{Timestamp: "1234567890.400000", Text: "message4"}},
		{Msg: slack.Msg{Timestamp: "1234567890.500000", Text: "message5"}},
	}},
	{Type: CMessages, ChannelID: TestChannelID, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.600000", Text: "Hello, again!"}},
		{Msg: slack.Msg{Timestamp: "1234567890.700000", Text: "And again!"}},
	}},
}

var testChunks = []Chunk{
	{Type: CChannelInfo, Timestamp: 123456, ChannelID: TestChannelID, Channel: &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: TestChannelID, NumMembers: 2}}}},
	{Type: CChannelUsers, Timestamp: 123456, ChannelID: TestChannelID, ChannelUsers: []string{"user1", "user2"}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.100000", Text: "message1"}},
		{Msg: slack.Msg{Timestamp: "1234567890.200000", Text: "message2"}},
		{Msg: slack.Msg{Timestamp: "1234567890.300000", Text: "message3"}},
		{Msg: slack.Msg{Timestamp: "1234567890.400000", Text: "message4"}},
		{Msg: slack.Msg{Timestamp: "1234567890.500000", Text: "message5"}},
	}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.600000", Text: "Hello, again!"}},
		{Msg: slack.Msg{Timestamp: "1234567890.700000", Text: "And again!"}},
	}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID, Messages: []slack.Message{
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
		ChannelID: TestChannelID,
		ThreadTS:  "1234567890.800000",
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
	{Type: CChannelInfo, Timestamp: 123456, ChannelID: TestChannelID2, Channel: &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: TestChannelID2, NumMembers: 2}}}},
	{Type: CChannelUsers, Timestamp: 123456, ChannelID: TestChannelID2, ChannelUsers: []string{"user3", "user4"}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID2, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.100000", Text: "message1"}},
		{Msg: slack.Msg{Timestamp: "1234567890.200000", Text: "message2"}},
		{Msg: slack.Msg{Timestamp: "1234567890.300000", Text: "message3"}},
		{Msg: slack.Msg{Timestamp: "1234567890.400000", Text: "message4"}},
		{Msg: slack.Msg{Timestamp: "1234567890.500000", Text: "message5"}},
	}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID2, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.600000", Text: "Hello, again!"}},
		{Msg: slack.Msg{Timestamp: "1234567890.700000", Text: "And again!"}},
	}},
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID2, Messages: []slack.Message{
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
		ChannelID: TestChannelID2,
		ThreadTS:  "1234567890.800000",
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
	{
		Type:        CSearchMessages,
		Timestamp:   1234567890,
		SearchQuery: "hello",
		SearchMessages: []slack.SearchMessage{
			{},
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
			want: []string{TestChannelID, TestChannelID2},
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
			p := &File{
				rs:  tt.fields.rs,
				idx: mkindex(tt.fields.rs),
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
				671:  offsetInfo{ID: TestChannelID, TS: 123456, Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				1506: offsetInfo{ID: TestChannelID, TS: 123456, Timestamps: []int64{1234567890600000, 1234567890700000}},
				1874: offsetInfo{ID: TestChannelID, TS: 123456, Timestamps: []int64{1234567890800000, 1234567890800000}},
				2330: offsetInfo{ID: "tC1234567890:1234567890.800000", Type: CThreadMessages, TS: 1234567890, Timestamps: []int64{1234567890900000, 1234567891100000}},
				3833: offsetInfo{ID: TestChannelID2, TS: 123456, Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				4667: offsetInfo{ID: TestChannelID2, TS: 123456, Timestamps: []int64{1234567890600000, 1234567890700000}},
				5034: offsetInfo{ID: TestChannelID2, TS: 123456, Timestamps: []int64{1234567890800000, 1234567890800000}},
				5489: offsetInfo{ID: "tC987654321:1234567890.800000", Type: CThreadMessages, TS: 1234567890, Timestamps: []int64{1234567890900000, 1234567891100000}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &File{
				rs:  tt.fields.rs,
				idx: mkindex(tt.fields.rs),
			}
			got, err := p.offsetTimestamps(t.Context())
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
		ots    offts
		chanID string
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
					596: offsetInfo{ID: TestChannelID, Timestamps: []int64{1234567890100000, 1234567890200000, 1234567890300000, 1234567890400000, 1234567890500000}},
				},
				chanID: TestChannelID,
			},
			want: map[int64]Addr{
				1234567890100000: {
					Offset: 596,
					Index:  0,
				},
				1234567890200000: {
					Offset: 596,
					Index:  1,
				},
				1234567890300000: {
					Offset: 596,
					Index:  2,
				},
				1234567890400000: {
					Offset: 596,
					Index:  3,
				},
				1234567890500000: {
					Offset: 596,
					Index:  4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, timeOffsets(tt.args.ots, tt.args.chanID))
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
		channelID string
		fn        func(ts time.Time, m *slack.Message) error
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
				channelID: TestChannelID,
				fn: func(ts time.Time, m *slack.Message) error {
					return nil
				},
			},
			wantFnCalls: []sortedArgs{
				{ts: time.Unix(1234567890, 100000000).UTC(), m: &testChunks[2].Messages[0]},
				{ts: time.Unix(1234567890, 200000000).UTC(), m: &testChunks[2].Messages[1]},
				{ts: time.Unix(1234567890, 300000000).UTC(), m: &testChunks[2].Messages[2]},
				{ts: time.Unix(1234567890, 400000000).UTC(), m: &testChunks[2].Messages[3]},
				{ts: time.Unix(1234567890, 500000000).UTC(), m: &testChunks[2].Messages[4]},
				{ts: time.Unix(1234567890, 600000000).UTC(), m: &testChunks[3].Messages[0]},
				{ts: time.Unix(1234567890, 700000000).UTC(), m: &testChunks[3].Messages[1]},
				{ts: time.Unix(1234567890, 800000000).UTC(), m: &testChunks[4].Messages[1]},
				{ts: time.Unix(1234567890, 900000000).UTC(), m: &testChunks[5].Messages[0]},
				{ts: time.Unix(1234567891, 100000000).UTC(), m: &testChunks[5].Messages[1]},
			},
			wantErr: false,
		},
		{
			name: "different channel",
			fields: fields{
				rs: marshalChunks(testChunks...),
			},
			args: args{
				channelID: "different",
				fn: func(ts time.Time, m *slack.Message) error {
					return nil
				},
			},
			wantFnCalls: []sortedArgs{},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &File{
				rs:  tt.fields.rs,
				idx: mkindex(tt.fields.rs),
			}
			var rec sortedArgsSlice

			recorder := func(ts time.Time, m *slack.Message) error {
				rec = append(rec, sortedArgs{ts, m})
				return nil
			}

			if err := p.Sorted(t.Context(), tt.args.channelID, false, recorder); (err != nil) != tt.wantErr {
				t.Errorf("File.Sorted() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantFnCalls.String(), rec.String())
		})
	}
}

func TestFile_Offsets(t *testing.T) {
	type fields struct {
		idx index
	}
	type args struct {
		id GroupID
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []int64
		want1  bool
	}{
		{
			name: "ok",
			fields: fields{
				idx: index{
					"1234567890": []int64{546},
					"1234567891": []int64{622},
				},
			},
			args: args{
				id: "1234567890",
			},
			want:  []int64{546},
			want1: true,
		},
		{
			name: "no entries",
			fields: fields{
				idx: index{
					"5555555555": []int64{},
				},
			},
			args: args{
				id: "5555555555",
			},
			want:  []int64{},
			want1: false,
		},
		{
			name: "does not exist",
			fields: fields{
				idx: index{
					"1234567890": []int64{546},
				},
			},
			args: args{
				id: "5555555555",
			},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				idx: tt.fields.idx,
			}
			got, got1 := f.offsets(tt.args.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("File.offsets() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("File.offsets() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFile_channelUsers(t *testing.T) {
	type fields struct {
		rs io.ReadSeeker
	}
	type args struct {
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			"returns users from the chunk",
			fields{
				rs: marshalChunks(testChunks...),
			},
			args{
				channelID: TestChannelID,
			},
			[]string{"user1", "user2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				rs:  tt.fields.rs,
				idx: mkindex(tt.fields.rs),
			}
			got, err := f.ChannelUsers(tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("File.channelUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("File.channelUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mkindex(rs io.ReadSeeker) index {
	idx, err := indexChunks(json.NewDecoder(rs))
	if err != nil {
		panic(err)
	}
	return idx
}

func TestFile_AllChannelInfos(t *testing.T) {
	c0wus := *testChunks[0].Channel
	c0wus.Members = testChunks[1].ChannelUsers

	c6wus := *testChunks[6].Channel
	c6wus.Members = testChunks[7].ChannelUsers

	type fields struct {
		rs io.ReadSeeker
	}
	tests := []struct {
		name    string
		fields  fields
		want    []slack.Channel
		wantErr bool
	}{
		{
			"captures all channels",
			fields{
				rs: marshalChunks(testChunks...),
			},
			[]slack.Channel{
				c0wus,
				c6wus,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				rs:  tt.fields.rs,
				idx: mkindex(tt.fields.rs),
			}
			got, err := f.AllChannelInfos()
			if (err != nil) != tt.wantErr {
				t.Errorf("File.AllChannelInfos() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFile_ChannelInfo(t *testing.T) {
	chanWithUsers := *testChunks[0].Channel
	chanWithUsers.Members = testChunks[1].ChannelUsers
	type args struct {
		channelID string
	}
	tests := []struct {
		name    string
		file    *File
		args    args
		want    *slack.Channel
		wantErr bool
	}{
		{
			name: "normal channel",
			file: &File{
				rs:  marshalChunks(testChunks...),
				idx: mkindex(marshalChunks(testChunks...)),
			},
			args: args{
				channelID: TestChannelID,
			},
			want:    &chanWithUsers,
			wantErr: false,
		},
		{
			name: "archived channel",
			file: &File{
				rs:  marshalChunks(archivedChannel...),
				idx: mkindex(marshalChunks(archivedChannel...)),
			},
			args: args{
				channelID: TestChannelID,
			},
			want:    archivedChannel[0].Channel,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.file
			got, err := f.ChannelInfo(tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("File.ChannelInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
