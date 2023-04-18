package chunk

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v2/internal/fixtures"
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
	type args struct {
		r io.ReadSeeker
	}
	tests := []struct {
		name    string
		args    args
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				r: marshalChunks(
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
			got, err := readChanInfo(tt.args.r)
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

// marshalChunks turns chunks into io.ReadSeeker
func marshalChunks(chunks ...Chunk) io.ReadSeeker {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	for _, c := range chunks {
		if err := enc.Encode(c); err != nil {
			panic(err)
		}
	}
	return bytes.NewReader(b.Bytes())
}
