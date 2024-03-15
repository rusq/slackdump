package viewer

import (
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
)

// Retriever is an interface for retrieving data from different sources.
type Retriever interface {
	// Name should return the name of the retriever underlying media, i.e.
	// directory or archive.
	Name() string
	// Channels should return all channels.
	Channels() ([]slack.Channel, error)
	// Users should return all users.
	Users() ([]slack.User, error)
	// AllMessages should return all messages for the given channel id.
	AllMessages(channelID string) ([]slack.Message, error)
	// AllThreadMessages should return all messages for the given tuple
	// (channelID, threadID).
	AllThreadMessages(channelID, threadID string) ([]slack.Message, error)
	// ChannelInfo should return the channel information for the given channel
	// id.
	ChannelInfo(channelID string) (*slack.Channel, error)
}

type ChunkRetriever struct {
	d *chunk.Directory
}

func NewChunkRetriever(d *chunk.Directory) *ChunkRetriever {
	return &ChunkRetriever{d: d}
}

func (c *ChunkRetriever) AllMessages(channelID string) ([]slack.Message, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.AllMessages(channelID)
}

func (c *ChunkRetriever) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.AllThreadMessages(channelID, threadID)
}

func (c *ChunkRetriever) ChannelInfo(channelID string) (*slack.Channel, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ChannelInfo(channelID)
}

func (c *ChunkRetriever) Channels() ([]slack.Channel, error) {
	return c.d.Channels()
}

func (c *ChunkRetriever) Name() string {
	return c.d.Name()
}

func (c *ChunkRetriever) Users() ([]slack.User, error) {
	return c.d.Users()
}
