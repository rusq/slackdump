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
)

// Chunk is a single chunk that was recorded.  It contains the type of chunk,
// the timestamp of the chunk, the channel ID, and the number of messages or
// files that were recorded.
type Chunk struct {
	// header
	Type      ChunkType `json:"t"`
	Timestamp int64     `json:"ts"`
	ChannelID string    `json:"id"`
	Count     int       `json:"n"` // number of messages or files

	// the rest
	IsThread bool `json:"r,omitempty"`
	// IsLast is set to true if this is the last chunk for the channel or
	// thread. Populated by Messages and ThreadMessages methods.
	IsLast bool `json:"l,omitempty"`
	// Number of threads in the message chunk.  Populated by Messages method.
	NumThreads int `json:"nt,omitempty"`

	// Channel contains the channel information.  It may not be immediately
	// followed by messages from the channel.  Populated by ChannelInfo method.
	Channel *slack.Channel `json:"ci,omitempty"`

	// Parent is populated in case the chunk is a thread, or a file. Populated
	// by ThreadMessages and Files methods.
	Parent *slack.Message `json:"p,omitempty"`
	// Messages contains a chunk of messages as returned by the API. Populated
	// by Messages and ThreadMessages methods.
	Messages []slack.Message `json:"m,omitempty"`
	// Files contains a chunk of files as returned by the API. Populated by
	// Files method.
	Files []slack.File `json:"f,omitempty"`

	// Users contains a chunk of users as returned by the API. Populated by
	// Users method.
	Users []slack.User `json:"u,omitempty"` // Populated by Users
	// Channels contains a chunk of channels as returned by the API. Populated
	// by Channels method.
	Channels []slack.Channel `json:"ch,omitempty"` // Populated by Channels
}

const (
	userChunkID    = "usr"
	channelChunkID = "chan"
)

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
	case CUsers:
		return userChunkID // static, one team per chunk file
	case CChannels:
		return channelChunkID // static, one team per chunk file.
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
