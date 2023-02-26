package chunk

import (
	"fmt"

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
)

// Chunk is a single chunk that was recorded.  It contains the type of chunk,
// the timestamp of the chunk, the channel ID, and the number of messages or
// files that were recorded.
type Chunk struct {
	Type            ChunkType       `json:"_t"`
	Timestamp       int64           `json:"_ts"`
	ChannelID       string          `json:"_id"`
	IsThreadMessage bool            `json:"_tm,omitempty"`
	Count           int             `json:"_c"` // number of messages or files
	Parent          *slack.Message  `json:"_p,omitempty"`
	Messages        []slack.Message `json:"_m,omitempty"`
	Files           []slack.File    `json:"_f,omitempty"`
}

func (c *Chunk) messageID() string {
	return c.ChannelID
}

func (c *Chunk) threadID() string {
	return threadID(c.ChannelID, c.Parent.ThreadTimestamp)
}

func threadID(channelID, threadTS string) string {
	return "t" + channelID + ":" + threadTS
}

// fileChunkID returns a unique ID for the file chunk.
func (c *Chunk) fileChunkID() string {
	return fileID(c.ChannelID, c.Parent.Timestamp)
}

func fileID(channelID, parentTS string) string {
	return "f" + channelID + ":" + parentTS
}

// ID returns a unique ID for the chunk.
func (c *Chunk) ID() string {
	switch c.Type {
	case CMessages:
		return c.messageID()
	case CThreadMessages:
		return c.threadID()
	case CFiles:
		return c.fileChunkID()
	}
	return fmt.Sprintf("<unknown:%d>", c.Type)
}
