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

func OpenExport(fsys fs.FS, name string) (*Export, error) {
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
		files:     NoStorage{},
		avatars:   NoStorage{},
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
		return OpenMattermostStorage(fsys)
	}
	idx, err := buildFileIndex(fsys, ".")
	if err != nil || len(idx) == 0 {
		return NoStorage{}, nil
	}
	return OpenStandardStorage(fsys, idx), nil
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

func (e *Export) Type() Flags {
	return FExport
}

// AllMessages returns all channel messages without thread messages.
func (e *Export) AllMessages(_ context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	it, err := e.walkChannelMessages(channelID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return func(yield func(slack.Message, error) bool) {
		for m, err := range it {
			if err != nil {
				yield(slack.Message{}, err)
				return
			}

			if m.ThreadTimestamp != "" && !structures.IsThreadStart(&m) {
				// skip thread messages
				continue
			}
			if !yield(m, nil) {
				return
			}
		}
	}, nil
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
				sm := slack.Message{Msg: *m.Msg}
				if !yield(sm, nil) {
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

func (e *Export) AllThreadMessages(_ context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	it, err := e.walkChannelMessages(channelID)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
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

func (e *Export) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	// doesn't matter, this method is used only in export conversion, and as
	// this is export it should never be called, just like your ex.
	panic("this method should never be called")
}

// ExportChanName returns the channel name, or the channel ID if it is a DM.
func ExportChanName(ch *slack.Channel) string {
	if ch.IsIM {
		return ch.ID
	}
	return ch.Name
}
