package export

import (
	"testing"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/slack-go/slack"
)

func TestIndex_Marshal(t *testing.T) {
	type args struct {
		fs fsadapter.FS
	}
	tests := []struct {
		name    string
		fields  index
		args    args
		wantErr bool
	}{
		{
			"x",
			index{Channels: []slack.Channel{{IsChannel: true}}},
			args{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := &index{
				Channels: tt.fields.Channels,
				Groups:   tt.fields.Groups,
				MPIMs:    tt.fields.MPIMs,
				DMs:      tt.fields.DMs,
				Users:    tt.fields.Users,
			}
			if err := idx.Marshal(tt.args.fs); (err != nil) != tt.wantErr {
				t.Errorf("Index.Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
