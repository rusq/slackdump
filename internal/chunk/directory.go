package chunk

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/rusq/slackdump/v2/internal/osext"
	"github.com/slack-go/slack"
)

const ext = ".json.gz"

// common filenames
const (
	FChannels  FileID = "channels"
	FUsers     FileID = "users"
	FWorkspace FileID = "workspace"
)

// Directory is an abstraction over the directory with chunk files.  It
// provides a way to write chunk files and read channels, users and messages
// across many the chunk files.  All functions that require a name, except
// functions with suffix RAW, will append an extension to the name
// automatically (".json.gz").  *RAW functions expect the full name of the
// file with the extension.  All files created by this package will be
// compressed with GZIP, unless stated otherwise.
type Directory struct {
	dir   string
	cache dcache

	wantCache bool
	fm        *filemgr
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
		dir:       dir,
		wantCache: true,
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

var errNoChannelInfo = errors.New("no channel info")

// Channels collects all channels from the chunk directory.  First, it
// attempts to find the channel.json.gz file, if it's not present, it will go
// through all conversation files and try to get "ChannelInfo" chunk from the
// each file.
func (d *Directory) Channels() ([]slack.Channel, error) {
	if val := d.cache.channels.Load(); val != nil {
		return val.([]slack.Channel), nil
	}
	// we are not using the channel.json.gz file because it doesn't contain
	// members.
	var ch []slack.Channel
	if err := filepath.WalkDir(d.dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ext) {
			return nil
		} else if de.IsDir() {
			return nil
		}
		chs, err := d.loadChanInfo(path)
		if err != nil {
			if errors.Is(err, errNoChannelInfo) {
				return nil
			}
			return err
		}
		ch = append(ch, chs...)
		return nil
	}); err != nil {
		return nil, err
	}
	d.cache.channels.Store(ch)
	return ch, nil
}

// Name returns the full directory path.
func (d *Directory) Name() string {
	return d.dir
}

func (d *Directory) loadChanInfo(fullpath string) ([]slack.Channel, error) {
	f, err := d.openRAW(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ch, err := d.readChanInfo(f)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

// readChanInfo returns the Channels from all the ChannelInfo chunks in the
// file.
func (d *Directory) readChanInfo(wf osext.ReadSeekCloseNamer) ([]slack.Channel, error) {
	cf, err := cachedFromReader(wf, d.wantCache)
	if err != nil {
		return nil, err
	}
	return cf.AllChannelInfos()
}

// loadChannelsJSON loads channels json file and returns a slice of
// slack.Channel.  It expects it to be GZIP compressed.
func (d *Directory) loadChannelsJSON(fullpath string) ([]slack.Channel, error) {
	f, err := d.openRAW(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cf, err := cachedFromReader(f, d.wantCache)
	if err != nil {
		return nil, err
	}
	return cf.AllChannels()
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
	return filepath.Join(d.dir, string(id)+ext)
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

// WorkspaceInfo returns the workspace info from the directory.  First it tries
// to find the workspace.json.gz file, if not found, it tries to get the info
// from users.json.gz and channels.json.gz.
func (d *Directory) WorkspaceInfo() (*slack.AuthTestResponse, error) {
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
