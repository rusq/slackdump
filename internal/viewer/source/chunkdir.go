package source

import (
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
)

type ChunkDir struct {
	d *chunk.Directory
}

func NewChunkDir(d *chunk.Directory) *ChunkDir {
	return &ChunkDir{d: d}
}

func (c *ChunkDir) AllMessages(channelID string) ([]slack.Message, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.AllMessages(channelID)
}

func (c *ChunkDir) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.AllThreadMessages(channelID, threadID)
}

func (c *ChunkDir) ChannelInfo(channelID string) (*slack.Channel, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ChannelInfo(channelID)
}

func (c *ChunkDir) Channels() ([]slack.Channel, error) {
	return c.d.Channels()
}

func (c *ChunkDir) Name() string {
	return c.d.Name()
}

func (c *ChunkDir) Users() ([]slack.User, error) {
	return c.d.Users()
}
