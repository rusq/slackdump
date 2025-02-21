package source

import (
	"context"
	"io/fs"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/testutil"
	"github.com/rusq/slackdump/v3/types"
)

func TestDump_Channels(t *testing.T) {
	type fields struct {
		c       []slack.Channel
		fs      fs.FS
		name    string
		Storage Storage
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "#455 skips attachments",
			fields: fields{
				fs: fixtures.FSTestDumpDir,
			},
			args: args{
				in0: context.Background(),
			},
			want: []slack.Channel{
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "CHY5HUESG",
						},
						Name: "everyone",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "test zip file",
			fields: fields{
				fs: fixtures.FSTestDumpZIP(t),
			},
			args: args{
				in0: context.Background(),
			},
			want: []slack.Channel{
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "CHY5HUESG",
						},
						Name: "everyone",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Dump{
				c:       tt.fields.c,
				fs:      tt.fields.fs,
				name:    tt.fields.name,
				Storage: tt.fields.Storage,
			}
			got, err := d.Channels(tt.args.in0)
			if (err != nil) != tt.wantErr {
				t.Errorf("Dump.Channels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_isDumpJSONFile(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "public channel",
			args: args{
				name: "C12345678.json",
			},
			want: true,
		},
		{
			name: "group conversation",
			args: args{
				name: "G12345678.json",
			},
			want: true,
		},
		{
			name: "DM",
			args: args{
				name: "D12345678.json",
			},
			want: true,
		},
		{
			name: "random",
			args: args{
				name: "random.json",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDumpJSONFile(tt.args.name); got != tt.want {
				t.Errorf("isDumpJSONFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertMessages(t *testing.T) {
	type args struct {
		cm []types.Message
	}
	tests := []struct {
		name string
		args args
		want []testutil.IterVal[slack.Message, error]
	}{
		{
			name: "empty",
			args: args{cm: []types.Message{}},
			want: []testutil.IterVal[slack.Message, error]{},
		},
		{
			name: "one",
			args: args{cm: []types.Message{
				{Message: slack.Message{Msg: slack.Msg{Text: "one"}}},
			}},
			want: []testutil.IterVal[slack.Message, error]{
				{T: slack.Message{Msg: slack.Msg{Text: "one"}}, U: nil},
			},
		},
		{
			name: "two",
			args: args{cm: []types.Message{
				{Message: slack.Message{Msg: slack.Msg{Text: "one"}}},
				{Message: slack.Message{Msg: slack.Msg{Text: "two"}}},
			}},
			want: []testutil.IterVal[slack.Message, error]{
				{T: slack.Message{Msg: slack.Msg{Text: "one"}}, U: nil},
				{T: slack.Message{Msg: slack.Msg{Text: "two"}}, U: nil},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := convertMessages(tt.args.cm)
			var i int
			for m, err := range it {
				assert.Equal(t, tt.want[i].T, m)
				assert.Equal(t, tt.want[i].U, err)
				i++
			}
		})
	}
}
