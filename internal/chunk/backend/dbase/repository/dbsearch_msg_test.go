package repository

import (
	"reflect"
	"testing"

	"github.com/rusq/slack"
)

var srchMsg1 = &slack.SearchMessage{
	Type: "message",
	Channel: slack.CtxChannel{
		ID:   "C123",
		Name: "chur",
	},
	User:      "U123",
	Username:  "bob",
	Timestamp: "1725318212.603879",
	Text:      "Hello, world!",
	Permalink: "http://slackdump.slack.com/archives/C123/p1725318212603879",
}

func TestNewDBSearchMessage(t *testing.T) {
	type args struct {
		chunkID int64
		idx     int
		sm      *slack.SearchMessage
	}
	tests := []struct {
		name    string
		args    args
		want    *DBSearchMessage
		wantErr bool
	}{
		{
			name: "creates a new DBSearchMessage",
			args: args{
				chunkID: 42,
				idx:     50,
				sm:      srchMsg1,
			},
			want: &DBSearchMessage{
				ID:          0, // autoincrement, handled by the database.
				ChunkID:     42,
				ChannelID:   "C123",
				ChannelName: ptr("chur"),
				TS:          "1725318212.603879",
				Text:        ptr("Hello, world!"),
				IDX:         50,
				Data:        must(marshal(srchMsg1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBSearchMessage(tt.args.chunkID, tt.args.idx, tt.args.sm)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBSearchMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBSearchMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
