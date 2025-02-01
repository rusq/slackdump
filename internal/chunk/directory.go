package chunk

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	chunkExt = ".json.gz"
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

	fm         *filemgr
	numWorkers int
	timestamp  int64

	wantCache bool
	readOnly  bool
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

func WithTimestamp(ts int64) DirOption {
	return func(d *Directory) {
		d.timestamp = ts
	}
}

func WithReadOnly() DirOption {
	return func(d *Directory) {
		d.readOnly = true
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
		timestamp:  time.Now().Unix(),
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
func CreateDir(dir string, opt ...DirOption) (*Directory, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return OpenDir(dir, opt...)
}

// RemoveAll deletes the directory and all its contents.  Make sure all files
// are closed.
func (d *Directory) RemoveAll() error {
	if d.readOnly {
		return nil
	}
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

type filereq struct {
	name string
	f    *File
}

func collectAll[T any](ctx context.Context, d *Directory, numwrk int, fn func(name string, f *File) ([]T, error)) ([]T, error) {
	var all []T
	fileC := make(chan filereq)
	errC := make(chan error, 1)
	go func() {
		defer close(fileC)
		defer close(errC)
		errC <- d.Walk(func(name string, f *File, err error) error {
			if err != nil {
				return err
			}
			fileC <- filereq{name: name, f: f}
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

// collectWorker collects the results by calling the function fn on each file
// from the fileC channel.  It sends the results to the resultsC channel.  It
// closes the file after the function is called.
func collectWorker[T any](fileC <-chan filereq, resultsC chan<- resultt[T], fn func(name string, f *File) ([]T, error)) {
	for f := range fileC {
		v, err := fn(f.name, f.f)
		resultsC <- resultt[T]{v, err}
		f.f.Close()
	}
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
		if !strings.HasSuffix(path, chunkExt) || de.IsDir() {
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

func (d *Directory) WalkVer(fn func(gid fileVersions, err error) error) error {
	return walkVersion(os.DirFS(d.dir), fn)
}

// WalkSync is the same as Walk, but it closes the file after the callback is
// called.
func (d *Directory) WalkSync(fn func(name string, f *File, err error) error) error {
	return d.Walk(func(name string, f *File, err error) error {
		defer f.Close()
		return fn(name, f, err)
	})
}

// Name returns the full directory path.
func (d *Directory) Name() string {
	return d.dir
}

func (d *Directory) Stat(id FileID) (fs.FileInfo, error) {
	return d.StatVersion(id, 0)
}

func (d *Directory) StatVersion(id FileID, ver int64) (fs.FileInfo, error) {
	return os.Stat(d.filever(id, ver))
}

// Users returns the collected users from the directory.
func (d *Directory) Users() ([]slack.User, error) {
	// versions are expected to be sorted ascending
	uv := &userVersion{Directory: d}
	return latestRec(os.DirFS(d.dir), uv, FUsers)
}

func (d *Directory) Channels(context.Context) ([]slack.Channel, error) {
	if val := d.cache.channels.Load(); val != nil {
		return val.([]slack.Channel), nil
	}
	var ch []slack.Channel
	fsys := os.DirFS(d.dir)
	if err := d.WalkVer(func(gid fileVersions, err error) error {
		if err != nil {
			return err
		}
		civ := &channelInfoVersion{Directory: d}
		cis, err := latestRec(fsys, civ, gid.ID)
		if err != nil {
			return err
		}
		ch = append(ch, cis...)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(ch, func(i, j int) bool {
		return ch[i].NameNormalized < ch[j].NameNormalized
	})
	d.cache.channels.Store(ch)
	return ch, nil
}

// Open opens a chunk file with the given name.  Extension is appended
// automatically.
func (d *Directory) Open(id FileID) (*File, error) {
	return d.OpenVersion(id, 0)
}

// OpenVersion opens a chunk file with the given name and version.  Extension
// is appended automatically.  You can discover the versions of the file with
// the [Versions] method.
func (d *Directory) OpenVersion(id FileID, ver int64) (*File, error) {
	filename := d.filever(id, ver)
	f, err := d.openRAW(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	return cachedFromReader(f, d.wantCache)
}

// Versions returns the versions of the chunk file id in the directory.
// each version is a timestamp of when the directory was updated, i.e. during
// consequent runs of "resume" command.
func (d *Directory) Versions(id FileID) ([]int64, error) {
	return AllVersions(os.DirFS(d.dir), id)
}

// filever returns the filename of the FileID with the given version. If ver is
// -1, it returns the filemask for the glob function to find all versions of
// the file.  If the version is 0, it returns the base file name (legacy
// behaviour).
func (d *Directory) filever(id FileID, ver int64) string {
	return filepath.Join(d.dir, filever(id, ver))
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
	if d.readOnly {
		return nil, os.ErrPermission
	}
	filename := d.filever(fileID, d.timestamp)
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
	wiv := &workspaceInfoVersion{Directory: d}
	wi, err := latestRec(os.DirFS(d.dir), wiv, FWorkspace, FUsers, FChannels)
	if err != nil {
		return nil, err
	}
	return wi[0], nil
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

func (d *Directory) AllMessages(channelID string) ([]slack.Message, error) {
	var mm structures.Messages
	err := d.WalkVer(func(gid fileVersions, err error) error {
		if err != nil {
			return err
		}
		mv := &messageVersion{Directory: d, ChannelID: channelID}
		m, err := latestRec(os.DirFS(d.dir), mv, gid.ID)
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

func (d *Directory) AllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	var mm structures.Messages
	var parent *slack.Message
	err := d.WalkVer(func(gid fileVersions, err error) error {
		if err != nil {
			return err
		}
		var (
			pmv = &parentMessageVersion{Directory: d, ChannelID: channelID, ThreadID: threadID}
			tmv = &threadMessageVersion{Directory: d, ChannelID: channelID, ThreadID: threadID}
		)
		if parent == nil {
			par, err := oneRec(os.DirFS(d.dir), pmv, gid.ID)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					return nil
				}
			} else {
				parent = &par
			}
		}
		rest, err := latestRec(os.DirFS(d.dir), tmv, gid.ID)
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

// FastAllThreadMessages returns all messages in the thread with the given id.  It assumes
// that the messages are in the same fileID versions.
func (d *Directory) FastAllThreadMessages(channelID, threadID string) ([]slack.Message, error) {
	var (
		pmv  = &parentMessageVersion{Directory: d, ChannelID: channelID, ThreadID: threadID}
		tmv  = &threadMessageVersion{Directory: d, ChannelID: channelID, ThreadID: threadID}
		fsys = os.DirFS(d.dir)
	)
	parent, err := oneRec(fsys, pmv, ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}
	rest, err := latestRec(fsys, tmv, ToFileID(channelID, "", false))
	if err != nil {
		return nil, err
	}

	return append([]slack.Message{parent}, rest...), nil
}

func (d *Directory) FastAllMessages(channelID string) ([]slack.Message, error) {
	mv := &messageVersion{Directory: d, ChannelID: channelID}
	mm, err := latestRec(os.DirFS(d.dir), mv, FileID(channelID))
	if err != nil {
		return nil, err
	}
	return mm, nil
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

func (d *Directory) ChannelInfo(id string) (*slack.Channel, error) {
	civ := &channelInfoVersion{Directory: d}
	cis, err := oneRec(os.DirFS(d.dir), civ, FileID(id))
	if err != nil {
		return nil, err
	}
	return &cis, nil
}

// FS returns the file system for the directory.
func (d *Directory) FS() fs.FS {
	return os.DirFS(d.dir)
}
