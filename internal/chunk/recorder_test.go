package chunk

import (
	"testing"

	"github.com/rusq/slack"
)

func TestEvent_ID(t *testing.T) {
	type fields struct {
		Type      ChunkType
		Timestamp int64
		ChannelID string
		ThreadTS  string
		Count     int
		Parent    *slack.Message
		Messages  []slack.Message
		Files     []slack.File
	}
	tests := []struct {
		name   string
		fields fields
		want   GroupID
	}{
		{
			"Message",
			fields{
				Type:      CMessages,
				ChannelID: "C123",
			},
			"C123",
		},
		{
			"Thread",
			fields{
				Type:      CThreadMessages,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{ThreadTimestamp: "123.456"},
				},
			},
			"tC123:123.456",
		},
		{
			"File",
			fields{
				Type:      CFiles,
				ChannelID: "C123",
				Parent: &slack.Message{
					Msg: slack.Msg{Timestamp: "123.456"},
				},
			},
			"fC123:123.456",
		},
		{
			"Unknown type",
			fields{
				Type:      ChunkType(255),
				ChannelID: "C123",
			},
			"<unknown:ChunkType(255)>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Chunk{
				Type:      tt.fields.Type,
				Timestamp: tt.fields.Timestamp,
				ChannelID: tt.fields.ChannelID,
				ThreadTS:  tt.fields.ThreadTS,
				Count:     tt.fields.Count,
				Parent:    tt.fields.Parent,
				Messages:  tt.fields.Messages,
				Files:     tt.fields.Files,
			}
			if got := e.ID(); got != tt.want {
				t.Errorf("Event.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}
