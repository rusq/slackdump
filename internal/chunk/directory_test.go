package chunk

import (
	"io"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/osext"
	"github.com/slack-go/slack"
)

// assortment of channel info chunks
var (
	TestPublicChannelInfo = Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:       "C01SPFM1KNY",
					IsShared: false,
				},
				Name:       "test",
				IsArchived: false,
			},
			IsChannel: true,
			IsMember:  true,
			IsGeneral: false,
		},
	}
	TestDMChannelInfo = Chunk{
		ChannelID: "D01MN4X7UGP",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:          "D01MN4X7UGP",
					IsOpen:      true,
					IsIM:        true,
					IsPrivate:   true,
					IsOrgShared: false,
				},
			},
		},
	}
)

// assortment of message chunks
var (
	TestPublicChannelMessages = Chunk{
		Type:      CMessages,
		ChannelID: "C01SPFM1KNY",
		Messages: []slack.Message{
			fixtures.Load[slack.Message](fixtures.TestMessage),
		},
	}
)

func Test_readChanInfo(t *testing.T) {
	dir := t.TempDir()
	type fields struct {
		wantCache bool
	}
	type args struct {
		r osext.ReadSeekCloseNamer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				wantCache: true,
			},
			args: args{
				r: testfilewrapper(
					filepath.Join(dir, "unit"),
					TestPublicChannelInfo,
					TestPublicChannelMessages,
				),
			},
			want: []slack.Channel{
				*TestPublicChannelInfo.Channel,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Directory{
				wantCache: tt.fields.wantCache,
			}
			got, err := d.readChanInfo(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readChanInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readChanInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testfilewrapper(name string, chunks ...Chunk) osext.ReadSeekCloseNamer {
	return nopCloser{
		ReadSeeker: marshalChunks(chunks...),
		name:       name,
	}
}

type nopCloser struct {
	name string
	io.ReadSeeker
}

func (n nopCloser) Close() error { return nil }

func (n nopCloser) Name() string { return n.name }

func TestOpenDir(t *testing.T) {

}
