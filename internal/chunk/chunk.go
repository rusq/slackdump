package chunk

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

// ChunkType is the type of chunk that was recorded.  There are three types:
// messages, thread messages, and files.
type ChunkType int

//go:generate stringer -type=ChunkType -trimprefix=C
const (
	CMessages ChunkType = iota
	CThreadMessages
	CFiles
	CUsers
	CChannels
	CChannelInfo
	CTeamInfo
)

// Chunk is a single chunk that was recorded.  It contains the type of chunk,
// the timestamp of the chunk, the channel ID, and the number of messages or
// files that were recorded.
type Chunk struct {
	Type      ChunkType `json:"_t"`
	Timestamp int64     `json:"_ts"`
	IsThread  bool      `json:"_tm,omitempty"`
	Count     int       `json:"_c"` // number of messages or files

	Channel *slack.Channel `json:"_ci,omitempty"`

	ChannelID string          `json:"_id"`
	Parent    *slack.Message  `json:"_p,omitempty"`
	Messages  []slack.Message `json:"_m,omitempty"`
	Files     []slack.File    `json:"_f,omitempty"`

	TeamID string       `json:"_tid,omitempty"`
	Team   *slack.Team  `json:"_ti,omitempty"`
	Users  []slack.User `json:"_u,omitempty"`
}

// ID returns a unique ID for the chunk.
func (c *Chunk) ID() string {
	switch c.Type {
	case CMessages:
		return c.ChannelID
	case CThreadMessages:
		return threadID(c.ChannelID, c.Parent.ThreadTimestamp)
	case CFiles:
		return id("f", c.ChannelID, c.Parent.Timestamp)
	case CChannelInfo:
		return channelInfoID(c.ChannelID, c.IsThread)
	case CTeamInfo:
		return id("ti", c.TeamID)
	case CUsers:
		return usersID(c.TeamID)
	case CChannels:
		return id("ch", c.TeamID)
	}
	return fmt.Sprintf("<unknown:%s>", c.Type)
}

func id(prefix string, ids ...string) string {
	return prefix + strings.Join(ids, ":")
}

func threadID(channelID, threadTS string) string {
	return id("t", channelID, threadTS)
}

func channelInfoID(channelID string, isThread bool) string {
	if isThread {
		return id("tci", channelID)
	}
	return id("ci", channelID)
}

func usersID(teamID string) string {
	return id("u", teamID)
}
