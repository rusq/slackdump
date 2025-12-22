package list

import (
	"testing"

	"github.com/rusq/slackdump/v3/types"
)

func Test_channels_Len(t *testing.T) {
	type fields struct {
		channels types.Channels
		users    types.Users
		opts     channelOptions
		common   commonOpts
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "zero",
			want: 0,
		},
		{
			name:   "two",
			fields: fields{channels: make(types.Channels, 2)},
			want:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &channels{
				channels: tt.fields.channels,
				users:    tt.fields.users,
				opts:     tt.fields.opts,
				common:   tt.fields.common,
			}
			if got := l.Len(); got != tt.want {
				t.Errorf("channels.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}
