package source

import (
	"context"
	"errors"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

type Dump struct {
	c     []slack.Channel
	fs    fs.FS
	name  string
	files Storage
}

func OpenDump(ctx context.Context, fsys fs.FS, name string) (*Dump, error) {
	var st Storage = fstNotFound{}
	if fst, err := NewDumpStorage(fsys); err == nil {
		st = fst
	}
	d := &Dump{
		fs:    fsys,
		name:  name,
		files: st,
	}
	// initialise channels for quick lookup
	c, err := d.Channels(ctx)
	if err != nil {
		return nil, err
	}
	d.c = c
	return d, nil
}

func (d Dump) Name() string {
	return d.name
}

func (d Dump) Type() string {
	return "dump"
}

func (d Dump) Channels(context.Context) ([]slack.Channel, error) {
	// if user was diligent enough to dump channels and save them in a file,
	// we can use that.
	if cc, err := unmarshal[[]slack.Channel](d.fs, "channels.json"); err == nil {
		return cc, nil
	}
	// this is highly inefficient: walking all files, reading their contents
	// and finding the channel names.  It is better to have a separate file
	// with the channel names and IDs.
	var cc []slack.Channel
	if err := fs.WalkDir(d.fs, ".", func(pth string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() && filepath.Base(pth) != "." { // skip all nested directories
			return fs.SkipDir
		}
		if !isDumpJSONFile(de.Name()) {
			return nil
		}

		c, err := unmarshalOne[types.Conversation](d.fs, pth)
		if err != nil {
			return err
		}
		cc = append(cc, slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID: c.ID,
				},
				Name: structures.NVL(c.Name, c.ID), // dump files do not have channel names for private conversations.
			},
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return cc, nil
}

func isDumpJSONFile(name string) bool {
	match, err := path.Match("[C|G|D]*.json", name)
	return err == nil && match
}

func (d Dump) Users(context.Context) ([]slack.User, error) {
	u, err := unmarshal[[]slack.User](d.fs, "users.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []slack.User{}, nil // user db not available
		}
		return nil, err
	}
	return u, nil
}

func (d Dump) AllMessages(_ context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	var cm []types.Message
	c, err := unmarshalOne[types.Conversation](d.fs, d.channelFile(channelID))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// we may be hitting a thread
		cm, err = d.threadHeadMessages(channelID)
		if err != nil {
			return nil, err
		}
	} else {
		cm = c.Messages
	}
	return convertMessages(cm), nil
}

func (d Dump) threadHeadMessages(channelID string) ([]types.Message, error) {
	// find all threads that belong to this channel that may have been
	// exported as separate files.
	files, err := fs.Glob(d.fs, d.threadFile(channelID, "*"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fs.ErrNotExist
	}
	// collect all thread start messages
	var cm []types.Message
	for _, f := range files {
		c, err := unmarshalOne[types.Conversation](d.fs, f)
		if err != nil {
			return nil, err
		}
		if len(c.Messages) == 0 {
			slog.Debug("no messages in file", "file", f)
			continue
		}
		// we only need the messages that start the threads.
		cm = append(cm, c.Messages[0])
	}
	types.SortMessages(cm)
	return cm, nil
}

func convertMessages(cm []types.Message) iter.Seq2[slack.Message, error] {
	iterFn := func(yield func(slack.Message, error) bool) {
		for _, m := range cm {
			if !yield(m.Message, nil) {
				return
			}
		}
	}
	return iterFn
}

func (d Dump) AllThreadMessages(_ context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	cm, err := d.findThreadInChannel(channelID, threadID)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		cm, err = d.findThreadFile(channelID, threadID)
		if err != nil {
			return nil, err
		}
	}
	return convertMessages(cm), nil
}

func (d Dump) channelFile(channelID string) string {
	return channelID + ".json"
}

func (d Dump) threadFile(channelID, threadID string) string {
	return channelID + "-" + threadID + ".json"
}

func (d Dump) findThreadInChannel(channelID, threadID string) ([]types.Message, error) {
	c, err := unmarshalOne[types.Conversation](d.fs, d.channelFile(channelID))
	if err != nil {
		return nil, err
	}
	for _, m := range c.Messages {
		if m.ThreadTimestamp == threadID {
			return m.ThreadReplies, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (d Dump) findThreadFile(channelID, threadID string) ([]types.Message, error) {
	c, err := unmarshalOne[types.Conversation](d.fs, d.threadFile(channelID, threadID))
	if err != nil {
		return nil, err
	}
	return c.Messages, nil
}

func (d Dump) ChannelInfo(_ context.Context, channelID string) (*slack.Channel, error) {
	for _, c := range d.c {
		if c.ID == channelID {
			return &c, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (d Dump) Close() error {
	return nil
}

func (d Dump) Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error) {
	return nil, errors.New("not supported yet")
}

func (d Dump) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	return nil, ErrNotSupported
}

func (d Dump) Files() Storage {
	return d.files
}

func (d Dump) Avatars() Storage {
	// Dump does not support avatars.
	return fstNotFound{}
}
