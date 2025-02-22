package source

import (
	"context"
	"errors"
	"fmt"
	"iter"
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
	d       *chunk.Directory
	fast    bool
	files   Storage
	avatars Storage
}

// NewChunkDir creates a new ChurkDir source.  It expects the attachments to be
// in the mattermost storage format.  If the attachments are not in the
// mattermost storage format, it will assume they were not downloaded.
func NewChunkDir(d *chunk.Directory, fast bool) *ChunkDir {
	rootFS := os.DirFS(d.Name())
	var stFile Storage = fstNotFound{}
	if fst, err := NewMattermostStorage(rootFS); err == nil {
		stFile = fst
	}
	var stAvatars Storage = fstNotFound{}
	if ast, err := NewAvatarStorage(rootFS); err == nil {
		stAvatars = ast
	}
	return &ChunkDir{d: d, files: stFile, avatars: stAvatars, fast: fast}
}

// AllMessages returns all messages for the channel.  Current restriction -
// it expects for all messages for the requested file to be in the file ID.json.gz.
// If messages for the channel are scattered across multiple file, it will not
// return all of them.
func (c *ChunkDir) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	var (
		mm  []slack.Message
		err error
	)
	if c.fast {
		mm, err = c.d.FastAllMessages(ctx, channelID)
	} else {
		mm, err = c.d.AllMessages(ctx, channelID)
	}
	if err != nil {
		return nil, err
	}
	return toIter(mm), nil
}

func toIter(mm []slack.Message) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		for _, m := range mm {
			if !yield(m, nil) {
				return
			}
		}
	}
}

func (c *ChunkDir) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	var (
		mm  []slack.Message
		err error
	)
	if c.fast {
		mm, err = c.d.FastAllThreadMessages(channelID, threadID)
	} else {
		mm, err = c.d.AllThreadMessages(ctx, channelID, threadID)
	}
	if err != nil {
		return nil, err
	}
	return toIter(mm), nil
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

func (c *ChunkDir) Users(context.Context) ([]slack.User, error) {
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

func (c *ChunkDir) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return c.d.WorkspaceInfo()
}

func (c *ChunkDir) Files() Storage {
	return c.files
}

func (c *ChunkDir) Avatars() Storage {
	return c.avatars
}

func (c *ChunkDir) Sorted(ctx context.Context, id string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	return c.d.Sorted(ctx, id, desc, cb)
}
