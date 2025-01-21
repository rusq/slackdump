package source

import (
	"os"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// ChunkDir is the chunk directory source.
//
// TODO: create an index of entries, otherwise it does the
// full scan of the directory.
type ChunkDir struct {
	d    *chunk.Directory
	fast bool
	Storage
}

// NewChunkDir creates a new ChurkDir source.  It expects the attachments to be
// in the mattermost storage format.  If the attachments are not in the
// mattermost storage format, it will assume they were not downloaded.
func NewChunkDir(d *chunk.Directory, fast bool) *ChunkDir {
	var st Storage = fstNotFound{}
	if fst, err := NewMattermostStorage(os.DirFS(d.Name())); err == nil {
		st = fst
	}
	return &ChunkDir{d: d, Storage: st, fast: fast}
}

// AllMessages returns all messages for the channel.  Current restriction -
// it expects for all messages for the requested file to be in the file ID.json.gz.
// If messages for the channel are scattered across multiple file, it will not
// return all of them.
func (c *ChunkDir) AllMessages(channelID string) ([]slack.Message, error) {
	if c.fast {
		return c.d.FastAllMessages(channelID)
	} else {
		return c.d.AllMessages(channelID)
	}
}

func (c *ChunkDir) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	if c.fast {
		return c.d.FastAllThreadMessages(channelID, threadID)
	}
	return c.d.AllThreadMessages(channelID, threadID)
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
