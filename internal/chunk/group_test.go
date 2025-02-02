package chunk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

// testIntersectingChunks is a test set of chunks that may have been a product of "resume" operation with
// intersecting timeframe.
var testIntersectingChunks = []Chunk{
	// intersecting chunks.
	{Type: CMessages, Timestamp: 123456, ChannelID: TestChannelID, Messages: []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.800000", Text: "And again!"}},
		{
			Msg: slack.Msg{
				ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa70",
				ThreadTimestamp: "1234567890.800000",
				Timestamp:       "1234567890.800000",
				Text:            "updated parent message",
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
				Text:            "updated parent message",
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
					Text:            "Hello, worldo!",
				},
			},
			{
				Msg: slack.Msg{
					ClientMsgID:     "ec821bf2-c241-471d-b511-967b6ed4aa72",
					Timestamp:       "1234567891.100000",
					ThreadTimestamp: "1234567890.123456",
					Text:            "Hello, Slacko!",
				},
			},
		},
	},
	// these are new chunks.
	{
		Type: CMessages, Timestamp: 1234567890, ChannelID: TestChannelID, Messages: []slack.Message{
			{Msg: slack.Msg{Timestamp: "2234567890.100000", Text: "Hello, world!"}},
			{Msg: slack.Msg{Timestamp: "2234567890.200000", Text: "Hello, world!"}},
			{Msg: slack.Msg{Timestamp: "2234567890.300000", Text: "Hello, world!"}},
			{Msg: slack.Msg{Timestamp: "2234567890.400000", Text: "Hello, world!"}},
		},
	},
}

func Test_filegroup_messageIndex(t *testing.T) {
	type args struct {
		ctx    context.Context
		chanID string
		desc   bool
	}
	tests := []struct {
		name string
		fg   filegroup
		args args
		want *grpMessageIndex
	}{
		{
			name: "returns index for TestChannelID",
			fg: filegroup{
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{
				ctx:    context.Background(),
				chanID: TestChannelID,
				desc:   false,
			},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{
					123456789_0100000: {idxFile: 0, addr: Addr{Offset: 671, Index: 0}},
					123456789_0200000: {idxFile: 0, addr: Addr{Offset: 671, Index: 1}},
					123456789_0300000: {idxFile: 0, addr: Addr{Offset: 671, Index: 2}},
					123456789_0400000: {idxFile: 0, addr: Addr{Offset: 671, Index: 3}},
					123456789_0500000: {idxFile: 0, addr: Addr{Offset: 671, Index: 4}},
					123456789_0600000: {idxFile: 0, addr: Addr{Offset: 1506, Index: 0}},
					123456789_0700000: {idxFile: 0, addr: Addr{Offset: 1506, Index: 1}},
					123456789_0800000: {idxFile: 0, addr: Addr{Offset: 1874, Index: 1}},
					123456789_0900000: {idxFile: 0, addr: Addr{Offset: 2330, Index: 0}},
					123456789_1100000: {idxFile: 0, addr: Addr{Offset: 2330, Index: 1}},
				},
				tsList: []int64{
					123456789_0100000,
					123456789_0200000,
					123456789_0300000,
					123456789_0400000,
					123456789_0500000,
					123456789_0600000,
					123456789_0700000,
					123456789_0800000,
					123456789_0900000,
					123456789_1100000,
				},
			},
		},
		{
			name: "returns index for TestChannelID in descending order",
			fg: filegroup{
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{
				ctx:    context.Background(),
				chanID: TestChannelID,
				desc:   true,
			},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{
					123456789_0100000: {idxFile: 0, addr: Addr{Offset: 671, Index: 0}},
					123456789_0200000: {idxFile: 0, addr: Addr{Offset: 671, Index: 1}},
					123456789_0300000: {idxFile: 0, addr: Addr{Offset: 671, Index: 2}},
					123456789_0400000: {idxFile: 0, addr: Addr{Offset: 671, Index: 3}},
					123456789_0500000: {idxFile: 0, addr: Addr{Offset: 671, Index: 4}},
					123456789_0600000: {idxFile: 0, addr: Addr{Offset: 1506, Index: 0}},
					123456789_0700000: {idxFile: 0, addr: Addr{Offset: 1506, Index: 1}},
					123456789_0800000: {idxFile: 0, addr: Addr{Offset: 1874, Index: 1}},
					123456789_0900000: {idxFile: 0, addr: Addr{Offset: 2330, Index: 0}},
					123456789_1100000: {idxFile: 0, addr: Addr{Offset: 2330, Index: 1}},
				},
				tsList: []int64{
					123456789_1100000,
					123456789_0900000,
					123456789_0800000,
					123456789_0700000,
					123456789_0600000,
					123456789_0500000,
					123456789_0400000,
					123456789_0300000,
					123456789_0200000,
					123456789_0100000,
				},
			},
		},
		{
			name: "index from same file different channel",
			fg: filegroup{
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{
				ctx:    context.Background(),
				chanID: TestChannelID2,
				desc:   false,
			},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{
					1234567890_100000: {idxFile: 0, addr: Addr{Offset: 3833, Index: 0}},
					1234567890_200000: {idxFile: 0, addr: Addr{Offset: 3833, Index: 1}},
					1234567890_300000: {idxFile: 0, addr: Addr{Offset: 3833, Index: 2}},
					1234567890_400000: {idxFile: 0, addr: Addr{Offset: 3833, Index: 3}},
					1234567890_500000: {idxFile: 0, addr: Addr{Offset: 3833, Index: 4}},
					1234567890_600000: {idxFile: 0, addr: Addr{Offset: 4667, Index: 0}},
					1234567890_700000: {idxFile: 0, addr: Addr{Offset: 4667, Index: 1}},
					1234567890_800000: {idxFile: 0, addr: Addr{Offset: 5034, Index: 1}},
					1234567890_900000: {idxFile: 0, addr: Addr{Offset: 5489, Index: 0}},
					1234567891_100000: {idxFile: 0, addr: Addr{Offset: 5489, Index: 1}},
				},
				tsList: []int64{
					1234567890_100000,
					1234567890_200000,
					1234567890_300000,
					1234567890_400000,
					1234567890_500000,
					1234567890_600000,
					1234567890_700000,
					1234567890_800000,
					1234567890_900000,
					1234567891_100000,
				},
			},
		},
		{
			name: "unrelated channel",
			fg: filegroup{
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{
				ctx:    context.Background(),
				chanID: "unrelated",
				desc:   false,
			},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{},
				tsList:  nil,
			},
		},
		{
			name: "two versions",
			fg: filegroup{
				// files are sorted from oldest to newest.
				&File{rs: marshalChunks(testIntersectingChunks...)},
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{
				ctx:    context.Background(),
				chanID: TestChannelID,
				desc:   false,
			},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{
					123456789_0100000: {idxFile: 1, addr: Addr{Offset: 671, Index: 0}},
					123456789_0200000: {idxFile: 1, addr: Addr{Offset: 671, Index: 1}},
					123456789_0300000: {idxFile: 1, addr: Addr{Offset: 671, Index: 2}},
					123456789_0400000: {idxFile: 1, addr: Addr{Offset: 671, Index: 3}},
					123456789_0500000: {idxFile: 1, addr: Addr{Offset: 671, Index: 4}},
					123456789_0600000: {idxFile: 1, addr: Addr{Offset: 1506, Index: 0}},
					123456789_0700000: {idxFile: 1, addr: Addr{Offset: 1506, Index: 1}},
					123456789_0800000: {idxFile: 0, addr: Addr{Offset: 0, Index: 1}},   // \
					123456789_0900000: {idxFile: 0, addr: Addr{Offset: 463, Index: 0}}, //  >these are in the original file as well
					123456789_1100000: {idxFile: 0, addr: Addr{Offset: 463, Index: 1}}, // /
					223456789_0100000: {idxFile: 0, addr: Addr{Offset: 1307, Index: 0}},
					223456789_0200000: {idxFile: 0, addr: Addr{Offset: 1307, Index: 1}},
					223456789_0300000: {idxFile: 0, addr: Addr{Offset: 1307, Index: 2}},
					223456789_0400000: {idxFile: 0, addr: Addr{Offset: 1307, Index: 3}},
				},
				tsList: []int64{
					123456789_0100000,
					123456789_0200000,
					123456789_0300000,
					123456789_0400000,
					123456789_0500000,
					123456789_0600000,
					123456789_0700000,
					123456789_0800000,
					123456789_0900000,
					123456789_1100000,
					223456789_0100000,
					223456789_0200000,
					223456789_0300000,
					223456789_0400000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// generate index for the group of files
			for i := range tt.fg {
				idx, err := indexChunks(json.NewDecoder(tt.fg[i].rs))
				if err != nil {
					t.Fatalf("failed to index chunks: %v", err)
				}
				tt.fg[i].idx = idx
			}
			got := tt.fg.messageIndex(tt.args.ctx, tt.args.chanID, tt.args.desc)
			assert.Equal(t, tt.want, got)
		})
	}
}
