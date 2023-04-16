// Package fixchunks contains chunk fixtures.
package fixchunks

import (
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/slack-go/slack"
)

// assortment of channel info chunks
var (
	TestPublicChannelInfo = chunk.Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      chunk.CChannelInfo,
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
	TestDMChannelInfo = chunk.Chunk{
		ChannelID: "D01MN4X7UGP",
		Type:      chunk.CChannelInfo,
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
	TestPublicChannelMessages = chunk.Chunk{
		Type:      chunk.CMessages,
		ChannelID: "C01SPFM1KNY",
		Messages: []slack.Message{
			fixtures.Load[slack.Message](fixtures.TestMessage),
		},
	}
)
