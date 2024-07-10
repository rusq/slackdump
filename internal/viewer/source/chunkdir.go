package source

import (
	"os"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
)

type ChunkDir struct {
	d *chunk.Directory
	filestorage
}

func NewChunkDir(d *chunk.Directory) *ChunkDir {
	var st filestorage = fstNotFound{}
	if fst, err := newMattermostStorage(os.DirFS(d.Name())); err == nil {
		st = fst
	}
	return &ChunkDir{d: d, filestorage: st}
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
	parent, err := f.ThreadParent(channelID, threadID)
	if err != nil {
		return nil, err
	}
	rest, err := f.AllThreadMessages(channelID, threadID)
	if err != nil {
		return nil, err
	}

	return append([]slack.Message{*parent}, rest...), nil
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

func (c *ChunkDir) Type() string {
	return "chunk"
}

func (c *ChunkDir) Users() ([]slack.User, error) {
	return c.d.Users()
}
