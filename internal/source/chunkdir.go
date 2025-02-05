package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
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

func (c *ChunkDir) ChannelInfo(_ context.Context, channelID string) (*slack.Channel, error) {
	f, err := c.d.Open(chunk.ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ChannelInfo(channelID)
}

func (c *ChunkDir) Channels(ctx context.Context) ([]slack.Channel, error) {
	return c.d.Channels(ctx)
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

func (c *ChunkDir) Close() error {
	return c.d.Close()
}

var ErrUnknownLinkType = errors.New("unknown link type")

func (c *ChunkDir) Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error) {
	l, err := c.d.Latest(ctx)
	if err != nil {
		return nil, err
	}
	mm := make(map[structures.SlackLink]time.Time, len(l))
	for k, v := range l {
		if ch, ok := k.AsChannelID(); ok {
			mm[structures.SlackLink{Channel: ch}] = v
		} else if ch, th, ok := k.AsThreadID(); ok {
			mm[structures.SlackLink{Channel: ch, ThreadTS: th}] = v
		} else {
			return nil, fmt.Errorf("%q: %w", k, ErrUnknownLinkType)
		}
	}
	return mm, nil
}

func (c *ChunkDir) WorkspaceInfo() (*slack.AuthTestResponse, error) {
	return c.d.WorkspaceInfo()
}
