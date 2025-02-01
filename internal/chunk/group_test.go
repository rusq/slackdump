package chunk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			name: "returns index",
			fg: filegroup{
				&File{rs: marshalChunks(testChunks...)},
			},
			args: args{ctx: context.Background(), desc: false},
			want: &grpMessageIndex{
				addrMsg: map[int64]grpAddr{
					123456789_0100000: {idxFile: 0, addr: Addr{Offset: 671, Index: 0}},
					123456789_0200000: {idxFile: 0, addr: Addr{Offset: 671, Index: 1}},
					123456789_0300000: {idxFile: 0, addr: Addr{Offset: 671, Index: 2}},
					123456789_0400000: {idxFile: 0, addr: Addr{Offset: 671, Index: 3}},
					123456789_0500000: {idxFile: 0, addr: Addr{Offset: 671, Index: 4}},
					123456789_0600000: {idxFile: 0, addr: Addr{Offset: 4667, Index: 0}},
					123456789_0700000: {idxFile: 0, addr: Addr{Offset: 4667, Index: 0}},
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
