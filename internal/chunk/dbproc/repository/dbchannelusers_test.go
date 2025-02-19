package repository

import (
	"reflect"
	"testing"
)

func TestNewDBChannelUser(t *testing.T) {
	type args struct {
		chunkID   int64
		n         int
		channelID string
		userID    string
	}
	tests := []struct {
		name    string
		args    args
		want    *DBChannelUser
		wantErr bool
	}{
		{
			name: "creates a new DBChannelUser",
			args: args{
				chunkID:   1,
				n:         50,
				channelID: "C100",
				userID:    "U100",
			},
			want: &DBChannelUser{
				ID:        "U100",
				ChunkID:   1,
				ChannelID: "C100",
				Index:     50,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBChannelUser(tt.args.chunkID, tt.args.n, tt.args.channelID, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBChannelUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBChannelUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
