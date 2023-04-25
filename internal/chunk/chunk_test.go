package chunk

import (
	"testing"

	"github.com/slack-go/slack"
)

func TestChunk_ID(t *testing.T) {
	type fields struct {
		Type      ChunkType
		Timestamp int64
		ThreadTS  string
		Count     int
		Channel   *slack.Channel
		ChannelID string
		Parent    *slack.Message
		Messages  []slack.Message
		Files     []slack.File
		Users     []slack.User
	}
	tests := []struct {
		name   string
		fields fields
		want   GroupID
	}{
		{
			name: "messages",
			fields: fields{
				Type:      CMessages,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
			},
			want: "C123",
		},
		{
			name: "threads",
			fields: fields{
				Type:      CThreadMessages,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{ThreadTimestamp: "1234"},
				},
			},
			want: "tC123:1234",
		},
		{
			name: "files",
			fields: fields{
				Type:      CFiles,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				Parent: &slack.Message{
					Msg: slack.Msg{Timestamp: "1234"},
				},
				ChannelID: "C123",
			},
			want: "fC123:1234",
		},
		{
			name: "channel info",
			fields: fields{
				Type:      CChannelInfo,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
			},
			want: "icC123",
		},
		{
			name: "channel info (thread)",
			fields: fields{
				Type:      CChannelInfo,
				Timestamp: 0,
				ThreadTS:  "1234",
				Count:     0,
				Channel:   nil,
				ChannelID: "C123",
			},
			want: "itC123",
		},
		{
			name: "users",
			fields: fields{
				Type:      CUsers,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "",
			},
			want: userChunkID,
		},
		{
			name: "channels",
			fields: fields{
				Type:      CChannels,
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
			},
			want: channelChunkID,
		},
		{
			name: "unknown",
			fields: fields{
				Type:      ChunkType(255),
				Timestamp: 0,
				Count:     0,
				Channel:   nil,
				ChannelID: "",
			},
			want: "<unknown:ChunkType(255)>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Chunk{
				Type:      tt.fields.Type,
				Timestamp: tt.fields.Timestamp,
				ThreadTS:  tt.fields.ThreadTS,
				Count:     tt.fields.Count,
				Channel:   tt.fields.Channel,
				ChannelID: tt.fields.ChannelID,
				Parent:    tt.fields.Parent,
				Messages:  tt.fields.Messages,
				Files:     tt.fields.Files,
				Users:     tt.fields.Users,
			}
			if got := c.ID(); got != tt.want {
				t.Errorf("Chunk.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}
