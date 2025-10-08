package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"log/slog"
	"path"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/export"
	"github.com/rusq/slackdump/v3/internal/chunk"
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
	cache     *threadCache
}

const cacheSz = 1 << 20

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
		cache:     newThreadCache(cacheSz),
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
	st, err := OpenStandardStorage(fsys)
	if err == nil {
		return st, nil
	}
	return NoStorage{}, nil
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

func (e *Export) buildThreadCache(ctx context.Context, channelID string) error {
	name, err := e.nameByID(channelID)
	if err != nil {
		return err
	}
	lg := slog.With("channel_id", channelID, "channel_name", name)
	if _, err := fs.Stat(e.fs, name); err != nil {
		return fmt.Errorf("%w: %s", fs.ErrNotExist, name)
	}
	lg.Debug("building thread cache")
	var n int
	if err := walkDir(e.fs, name, func(file string) error {
		if err := yieldFileContents(ctx, e.fs, file, func(m slack.Message, err error) bool {
			if err != nil {
				return false
			}
			if (structures.IsThreadStart(&m) && !structures.IsEmptyThread(&m)) || structures.IsThreadMessage(&m.Msg) {
				if err := e.cache.Update(ctx, name, m.ThreadTimestamp, file); err != nil {
					slog.ErrorContext(ctx, "error updating cache", "error", err)
				}
				n++
			}
			return true
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	slog.DebugContext(ctx, "caching completed", "thread_count", n)
	return nil
}

// AllMessages returns all channel messages without thread messages.
func (e *Export) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	if err := e.buildThreadCache(ctx, channelID); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	it, err := e.walkChannelMessages(ctx, channelID)
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

// yieldFileContents is meant to work with export json files and will call
// yield function for every message in the file. It is meant to work with
// fs.WalkDir function, when the yield function returns false (stop iteration),
// it returns `fs.SkipAll` error.  If calling this function not from the Walk
// function, this error indicates that file iteration should stop.
func yieldFileContents(ctx context.Context, fsys fs.FS, file string, yield func(slack.Message, error) bool) error {
	em, err := unmarshal[[]export.ExportMessage](fsys, file)
	if err != nil {
		var jsonErr *json.SyntaxError
		if errors.As(err, &jsonErr) {
			slog.WarnContext(ctx, "skipping broken file", "pth", file, "err", err)
			return nil
		}
		return err
	}
	for i, m := range em {
		if m.Msg == nil {
			slog.DebugContext(ctx, "skipping an empty message", "pth", file, "index", i)
			continue
		}
		sm := slack.Message{Msg: *m.Msg}
		if !yield(sm, nil) {
			return fs.SkipAll
		}
	}
	return nil
}

// fullScanIter is the message iterator that always scans all messages and
// populates the cache with discovered threads.
type fullScanIter struct {
	ctx  context.Context
	name string
	fs   fs.FS
}

func newFullScanIter(ctx context.Context, fs fs.FS, chanName string) *fullScanIter {
	return &fullScanIter{
		ctx:  ctx,
		name: chanName,
		fs:   fs,
	}
}

func walkDir(fsys fs.FS, name string, cb func(file string) error) error {
	err := fs.WalkDir(fsys, name, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && file != name {
			return fs.SkipDir
		}
		if path.Ext(file) != ".json" {
			return nil
		}
		return cb(file)
	})
	return err
}

// Iter iterates through all messages for the given channel name. It
// updates the cache with discovered threads.
func (w *fullScanIter) Iter(yield func(slack.Message, error) bool) {
	ctx, task := trace.NewTask(w.ctx, "full_scan_iter")
	defer task.End()
	err := walkDir(w.fs, w.name, func(file string) error {
		if err := yieldFileContents(ctx, w.fs, file, yield); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		yield(slack.Message{}, err)
	}
}

// fileListIter is meant to reduce the scope of iteration to the given file
// list.
type fileListIter struct {
	ctx   context.Context
	fs    fs.FS
	files []string
}

func (w *fileListIter) Iter(yield func(slack.Message, error) bool) {
	ctx, task := trace.NewTask(w.ctx, "file_list_iter")
	defer task.End()
	for _, file := range w.files {
		if err := yieldFileContents(ctx, w.fs, file, yield); err != nil {
			if errors.Is(err, fs.SkipAll) {
				return
			}
			yield(slack.Message{}, err)
			return
		}
	}
}

func (e *Export) nameByID(channelID string) (string, error) {
	name, ok := e.chanNames[channelID]
	if !ok {
		return "", fmt.Errorf("%w: %s", fs.ErrNotExist, channelID)
	}
	return name, nil
}

func (e *Export) walkChannelMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	name, err := e.nameByID(channelID)
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(e.fs, name); err != nil {
		return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
	}
	fsi := newFullScanIter(ctx, e.fs, name)
	return fsi.Iter, nil
}

var errNotInCache = errors.New("channel not in cache")

func (e *Export) walkCachedThreads(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	name, err := e.nameByID(channelID)
	if err != nil {
		return nil, err
	}
	if !e.cache.Exists(name) {
		return nil, fmt.Errorf("channel: %w", errNotInCache)
	}
	// get all files for the thread.
	files, ok := e.cache.Get(name, threadID)
	if !ok {
		return nil, fmt.Errorf("thread: %w", errNotInCache)
	}
	fli := fileListIter{ctx, e.fs, files}
	return fli.Iter, err
}

// AllThreadMessages is highly suboptimal. For each thread it does a full scan of the channel
// json files until it finds the thread every time it is called. There's no caching.
//
// TODO: index all thread messages during initial walking build a map of channel:thread->json file.
func (e *Export) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
	// try cached first
	lg := slog.With("channel_id", channelID, "thread_ts", threadID)
	it, err := e.walkCachedThreads(ctx, channelID, threadID)
	if err != nil {
		if !errors.Is(err, errNotInCache) {
			return nil, err
		}
		lg.DebugContext(ctx, "cache not available, initiating full scan")
		it, err = e.walkChannelMessages(ctx, channelID)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, ErrNotFound
			}
			return nil, err
		}
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
	// there will be no resume on export.
	return nil, ErrNotSupported
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
