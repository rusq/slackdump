package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"path"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/export"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// Export implements viewer.Sourcer for the zip file Slack export format.
type Export struct {
	fs        fs.FS
	channels  []slack.Channel
	chanNames map[string]string // maps the channel id to the channel name.
	name      string            // name of the file
	idx       structures.ExportIndex
	files     Storage
	avatars   Storage
}

func NewExport(fsys fs.FS, name string) (*Export, error) {
	var idx structures.ExportIndex
	if err := idx.Unmarshal(fsys); err != nil {
		return nil, err
	}
	chans := idx.Restore()
	z := &Export{
		fs:        fsys,
		name:      name,
		idx:       idx,
		channels:  chans,
		chanNames: make(map[string]string, len(chans)),
		files:     fstNotFound{},
		avatars:   fstNotFound{},
	}
	// initialise channels for quick lookup
	for _, ch := range z.channels {
		z.chanNames[ch.ID] = structures.NVL(ch.Name, ch.ID)
	}
	// determine files path
	fst, err := loadStorage(fsys)
	if err != nil {
		return nil, err
	}
	z.files = fst
	if fst, err := NewAvatarStorage(fsys); err == nil {
		z.avatars = fst
	}

	return z, nil
}

// loadStorage determines the type of the file storage used and initialises
// appropriate Storage implementation.
func loadStorage(fsys fs.FS) (Storage, error) {
	if _, err := fs.Stat(fsys, chunk.UploadsDir); err == nil {
		return NewMattermostStorage(fsys)
	}
	idx, err := buildFileIndex(fsys, ".")
	if err != nil || len(idx) == 0 {
		return fstNotFound{}, nil
	}
	return NewStandardStorage(fsys, idx), nil
}

func (e *Export) Channels(context.Context) ([]slack.Channel, error) {
	return e.channels, nil
}

func (e *Export) Users(context.Context) ([]slack.User, error) {
	return e.idx.Users, nil
}

func (e *Export) Close() error {
	return nil
}

func (e *Export) Name() string {
	return e.name
}

func (e *Export) Type() string {
	return "export"
}

// AllMessages returns all channel messages without thread messages.
func (e *Export) AllMessages(_ context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	return e.walkChannelMessages(channelID)
}

func (e *Export) walkChannelMessages(channelID string) (iter.Seq2[slack.Message, error], error) {
	name, ok := e.chanNames[channelID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, channelID)
	}
	_, err := fs.Stat(e.fs, name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
	}
	iterFn := func(yield func(slack.Message, error) bool) {
		err := fs.WalkDir(e.fs, name, func(pth string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || path.Ext(pth) != ".json" {
				return nil
			}
			// read the file
			em, err := unmarshal[[]export.ExportMessage](e.fs, pth)
			if err != nil {
				return err
			}
			for i, m := range em {
				if m.Msg == nil {
					slog.Default().Debug("skipping an empty message", "pth", pth, "index", i)
					continue
				}
				if !yield(slack.Message{Msg: *m.Msg}, nil) {
					return fs.SkipAll
				}
			}
			return nil
		})
		if err != nil {
			yield(slack.Message{}, err)
		}
	}
	return iterFn, nil
}

func isThreadMessage(m *slack.Msg) bool {
	return m.ThreadTimestamp != "" && m.ThreadTimestamp != m.Timestamp
}

func (e *Export) AllThreadMessages(_ context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	it, err := e.walkChannelMessages(channelID)
	if err != nil {
		return nil, err
	}
	iterFn := func(yield func(slack.Message, error) bool) {
		for m, err := range it {
			if err != nil {
				yield(slack.Message{}, err)
				return
			}
			if m.ThreadTimestamp == threadID {
				if !yield(m, nil) {
					return
				}
			}
		}
	}
	return iterFn, nil
}

func (e *Export) ChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	c, err := e.Channels(ctx)
	if err != nil {
		return nil, err
	}
	for _, ch := range c {
		if ch.ID == channelID {
			return &ch, nil
		}
	}
	return nil, fmt.Errorf("%s: %s", "channel not found", channelID)
}

func (e *Export) Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error) {
	return nil, errors.New("not supported yet")
}

func (e *Export) WorkspaceInfo(context.Context) (*slack.AuthTestResponse, error) {
	// potentially the URL of the workspace is contained in file attachments, but until
	// AllMessages is implemented with iterators, it's too expensive to get.
	return nil, ErrNotSupported
}

func (e *Export) Files() Storage {
	return e.files
}

func (e *Export) Avatars() Storage {
	return e.avatars
}
