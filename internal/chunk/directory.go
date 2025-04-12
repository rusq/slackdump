package chunk

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/osext"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// file extensions
const (
	ChunkExt = ".json.gz"
	extIdx   = ".idx"
)

// common filenames
const (
	FChannels  FileID = "channels"
	FUsers     FileID = "users"
	FWorkspace FileID = "workspace"
	FSearch    FileID = "search"
)

const (
	UploadsDir = "__uploads" // for serving files
	AvatarsDir = "__avatars"
)

// Directory is an abstraction over the directory with chunk files.  It
// provides a way to write chunk files and read channels, users and messages
// across many the chunk files.  All functions that require a name, except
// functions with suffix RAW, will append an extension to the name
// automatically (".json.gz").  *RAW functions expect the full name of the
// file with the extension.  All files created by this package will be
// compressed with GZIP, unless stated otherwise.
type Directory struct {
	// dir is a path to a physical directory on the filesystem with chunks and
	// uploads.
	dir   string
	cache dcache

	wantCache  bool
	fm         *filemgr
	numWorkers int
}

type dcache struct {
	channels atomic.Value // []slack.Channel
}

type DirOption func(*Directory)

func WithCache(enabled bool) DirOption {
	return func(d *Directory) {
		d.wantCache = enabled
	}
}

func WithNumWorkers(n int) DirOption {
	return func(d *Directory) {
		d.numWorkers = n
	}
}

// OpenDir "opens" an existing directory for read and write operations.
// It expects the directory to exist and to be a directory, otherwise it will
// return an error.
func OpenDir(dir string, opt ...DirOption) (*Directory, error) {
	if fi, err := os.Stat(dir); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	d := &Directory{
		dir:        dir,
		wantCache:  true,
		numWorkers: 16,
	}
	for _, o := range opt {
		o(d)
	}
	if d.wantCache {
		fm, err := newFileMgr()
		if err != nil {
			return nil, err
		}
		d.fm = fm
	}
	return d, nil
}

// CreateDir creates and opens a directory.  It will create all parent
// directories if they don't exist.
func CreateDir(dir string) (*Directory, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return OpenDir(dir)
}

// RemoveAll deletes the directory and all its contents.  Make sure all files
// are closed.
func (d *Directory) RemoveAll() error {
	_ = d.Close()
	return os.RemoveAll(d.dir)
}

// Close closes the directory and all open files.
func (d *Directory) Close() error {
	if d.fm != nil {
		return d.fm.Destroy()
	}
	return nil
}

type resultt[T any] struct {
	v   []T
	err error
}

func collectAll[T any](ctx context.Context, d *Directory, numwrk int, fn func(*File) ([]T, error)) ([]T, error) {
	var all []T
	fileC := make(chan *File)
	errC := make(chan error, 1)
	go func() {
		defer close(fileC)
		defer close(errC)
		errC <- d.Walk(func(name string, f *File, err error) error {
			if err != nil {
				return err
			}
			fileC <- f
			return nil
		})
	}()

	resultsC := make(chan resultt[T])
	var wg sync.WaitGroup
	wg.Add(numwrk)
	for range numwrk {
		go func() {
			collectWorker(fileC, resultsC, fn)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(resultsC)
	}()

LOOP:
	for {
		select {
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		case res, more := <-resultsC:
			if !more {
				break LOOP
			}
			if res.err != nil {
				return nil, res.err
			}
			all = append(all, res.v...)
		}
	}
	if err := <-errC; err != nil {
		return nil, err
	}
	return all, nil
}

func collectWorker[T any](fileC <-chan *File, resultsC chan<- resultt[T], fn func(*File) ([]T, error)) {
	for f := range fileC {
		v, err := fn(f)
		resultsC <- resultt[T]{v, err}
		f.Close()
	}
}

// Channels collects all channels from the chunk directory.  First, it attempts
// to find the channel.json.gz file, if it's not present, it will go through
// all conversation files and try to get "ChannelInfo" chunk from each file.
func (d *Directory) Channels(ctx context.Context) ([]slack.Channel, error) {
	if val := d.cache.channels.Load(); val != nil {
		slog.Debug("channels: cache hit")
		return val.([]slack.Channel), nil
	}
	slog.Debug("channels: cache miss")
	ch, err := collectAll(ctx, d, d.numWorkers, func(f *File) ([]slack.Channel, error) {
		c, err := f.AllChannelInfos()
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return c, nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(ch, func(i, j int) bool {
		return ch[i].NameNormalized < ch[j].NameNormalized
	})

	d.cache.channels.Store(ch)
	return ch, nil
}

type result struct {
	ci  []slack.Channel
	err error
}

// Walk iterates over all chunk files in the directory and calls the function
// for each file.  If the function returns an error, the iteration stops.
// It does not close files after the callback is called, so it's a caller's
// responsibility to close it.
func (d *Directory) Walk(fn func(name string, f *File, err error) error) error {
	return filepath.WalkDir(d.dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		var (
			isSupported = strings.HasSuffix(path, ChunkExt)
			isDir       = de.IsDir()
			isHidden    = len(de.Name()) > 0 && de.Name()[0] == '.'
		)
		if !isSupported || isDir || isHidden {
			return nil
		}
		f, err := d.openRAW(path)
		if err != nil {
			return fn(path, nil, err)
		}
		cf, err := cachedFromReader(f, d.wantCache)
		return fn(path, cf, err)
	})
}

// WalkSync is the same as Walk, but it closes the file after the callback is
// called.
func (d *Directory) WalkSync(fn func(name string, f *File, err error) error) error {
	return d.Walk(func(name string, f *File, err error) error {
		if err != nil {
			return err
		}
		defer f.Close()
		return fn(name, f, nil)
	})
}

// Name returns the full directory path.
func (d *Directory) Name() string {
	return d.dir
}

func (d *Directory) Stat(id FileID) (fs.FileInfo, error) {
	return os.Stat(d.filename(id))
}

// Users returns the collected users from the directory.
func (d *Directory) Users() ([]slack.User, error) {
	f, err := d.Open(FUsers)
	if err != nil {
		return nil, fmt.Errorf("unable to open users file %q: %w", d.filename(FUsers), err)
	}
	defer f.Close()
	users, err := f.AllUsers()
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Open opens a chunk file with the given name.  Extension is appended
// automatically.
func (d *Directory) Open(id FileID) (*File, error) {
	f, err := d.openRAW(d.filename(id))
	if err != nil {
		return nil, err
	}
	return cachedFromReader(f, d.wantCache)
}

// OpenRAW opens a compressed chunk file with filename within the directory,
// and returns a ReadSeekCloser.  filename is the full name of the file with
// extension.
func (d *Directory) OpenRAW(filename string) (io.ReadSeekCloser, error) {
	return d.openRAW(filename)
}

func (d *Directory) openRAW(filename string) (osext.ReadSeekCloseNamer, error) {
	if d.wantCache {
		return d.fm.Open(filename)
	}
	return openChunks(filename)
}

// openChunks opens an existing chunk file and returns a ReadSeekCloser.  It
// expects a chunkfile to be a gzip-compressed file.
func openChunks(filename string) (osext.ReadSeekCloseNamer, error) {
	f, err := openfile(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tf, err := osext.UnGZIP(f)
	if err != nil {
		return nil, err
	}

	return osext.RemoveOnClose(tf), nil
}

func openfile(filename string) (*os.File, error) {
	if fi, err := os.Stat(filename); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("chunk file is a directory")
	} else if fi.Size() == 0 {
		return nil, errors.New("chunk file is empty")
	}
	return os.Open(filename)
}

// filename returns the full path of the chunk file with the given fileID.
func (d *Directory) filename(id FileID) string {
	return filepath.Join(d.dir, string(id)+ChunkExt)
}

// Create creates the chunk file with the given name.  Extension is appended
// automatically.
//
// Example:
//
//	cd, _ := chunk.OpenDirectory("chunks")
//	f, _ := cd.Create("channels") // creates channels.json.gz
//
// It will NOT overwrite an existing file and will return an error if the file
// exists.
func (d *Directory) Create(fileID FileID) (io.WriteCloser, error) {
	filename := d.filename(fileID)
	if fi, err := os.Stat(filename); err == nil {
		if fi.IsDir() {
			return nil, fmt.Errorf("is a directory: %s", filename)
		}
		if fi.Size() > 0 {
			return nil, fmt.Errorf("file %s exists and not empty", filename)
		}
	}
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	gz := gzip.NewWriter(f)
	return &closewrapper{WriteCloser: gz, underlying: f}, nil
}

type closewrapper struct {
	io.WriteCloser
	underlying io.Closer
}

func (c *closewrapper) Close() error {
	if err := c.WriteCloser.Close(); err != nil {
		return err
	}
	return c.underlying.Close()
}

// WorkspaceInfo returns the workspace info from the directory.
func (d *Directory) WorkspaceInfo() (*slack.AuthTestResponse, error) {
	//  First it tries to find the workspace.json.gz file, if not found,
	// it tries to get the info from users.json.gz and channels.json.gz.
	for _, name := range []FileID{FWorkspace, FUsers, FChannels} {
		f, err := d.Open(name)
		if err != nil {
			continue
		}
		defer f.Close()
		wi, err := f.WorkspaceInfo()
		if err != nil {
			continue
		}
		return wi, nil
	}
	return nil, errors.New("no workspace info found")
}

func cachedFromReader(wf osext.ReadSeekCloseNamer, wantCache bool) (*File, error) {
	if !wantCache {
		return FromReader(wf)
	}
	// check if index exists.  If it does, read it and return chunk.File with it.
	r, err := os.Open(wf.Name() + extIdx)
	if err == nil {
		defer r.Close()
		dec := gob.NewDecoder(r)
		var idx index
		if err := dec.Decode(&idx); err != nil {
			return nil, err
		}
		return fromReaderWithIndex(wf, idx)
	}
	// write index
	cf, err := FromReader(wf)
	if err != nil {
		return nil, err
	}
	w, err := os.Create(wf.Name() + extIdx)
	if err != nil {
		return nil, err
	}
	defer w.Close()
	enc := gob.NewEncoder(w)
	if err := enc.Encode(cf.idx); err != nil {
		return nil, err
	}
	return cf, nil
}

// File returns the file with the given id and name.
func (d *Directory) File(id string, name string) (fs.File, error) {
	return os.Open(filepath.Join(d.dir, UploadsDir, id, name))
}

func (d *Directory) AllMessages(ctx context.Context, channelID string) ([]slack.Message, error) {
	var mm structures.Messages
	err := d.WalkSync(func(name string, f *File, err error) error {
		if err != nil {
			return err
		}
		m, err := f.AllMessages(ctx, channelID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil
			}
			return err
		}
		mm = append(mm, m...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Sort(mm)
	return mm, nil
}

func (d *Directory) AllThreadMessages(_ context.Context, channelID, threadID string) ([]slack.Message, error) {
	var mm structures.Messages
	var parent *slack.Message
	err := d.WalkSync(func(name string, f *File, err error) error {
		if err != nil {
			return err
		}
		if parent == nil {
			par, err := f.ThreadParent(channelID, threadID)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					return err
				}
			} else {
				parent = par
			}
		}
		rest, err := f.AllThreadMessages(channelID, threadID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		mm = append(mm, rest...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("parent not found for channel: %s, thread: %s", channelID, threadID)
	}
	sort.Sort(mm)
	return append([]slack.Message{*parent}, mm...), nil
}

func (d *Directory) FastAllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	// try open the thread file
	fileID := ToFileID(channelID, threadID, true)
	_, err := d.Stat(fileID)
	if err != nil {
		fileID = ToFileID(channelID, threadID, false)
	}
	f, err := d.Open(fileID)
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

func (d *Directory) FastAllMessages(ctx context.Context, channelID string) ([]slack.Message, error) {
	f, err := d.Open(FileID(channelID))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.AllMessages(ctx, channelID)
}

// Latest returns the latest timestamps for the channels and threads
// in the directory.
func (d *Directory) Latest(ctx context.Context) (map[GroupID]time.Time, error) {
	latest := make(map[GroupID]time.Time)
	err := d.WalkSync(func(name string, f *File, err error) error {
		if err != nil {
			return err
		}
		li, err := f.Latest(ctx)
		if err != nil {
			return err
		}
		for k, v := range li {
			current, ok := latest[k]
			if !ok || v.After(current) {
				latest[k] = v
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}

func (d *Directory) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	// TODO: this is oversimplification.  The messages for the channel in
	// canonical chunk directory may be stored in multiple files.
	f, err := d.Open(FileID(channelID))
	if err != nil {
		return err
	}
	return f.Sorted(ctx, channelID, desc, cb)
}

// ToChunk writes all chunks from the directory to the encoder.
func (d *Directory) ToChunk(ctx context.Context, enc Encoder, _ int64) error {
	err := d.WalkSync(func(name string, f *File, err error) error {
		if err != nil {
			return err
		}
		if err := f.ForEach(func(ch *Chunk) error {
			return enc.Encode(ctx, ch)
		}); err != nil {
			return err
		}
		return nil
	})
	return err
}
