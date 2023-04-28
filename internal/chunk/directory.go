package chunk

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/osext"
)

const ext = ".json.gz"

// common filenames
const (
	FChannels  = "channels"
	FUsers     = "users"
	FWorkspace = "workspace"
)

// Directory is an abstraction over the directory with chunk files.  It
// provides a way to write chunk files and read channels, users and messages
// across many the chunk files.  All functions that require a name, except
// functions with suffix RAW, will append an extension to the name
// automatically (".json.gz").  *RAW functions expect the full name of the
// file with the extension.  All files created by this package will be
// compressed with GZIP, unless stated otherwise.
type Directory struct {
	dir string
}

// OpenDir "opens" an existing directory for read and write operations.
// It expects the directory to exist and to be a directory, otherwise it will
// return an error.
func OpenDir(dir string) (*Directory, error) {
	if fi, err := os.Stat(dir); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}
	return &Directory{dir: dir}, nil
}

// CreateDir creates and opens a directory.  It will create all parent
// directories if they don't exist.
func CreateDir(dir string) (*Directory, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Directory{dir: dir}, nil
}

// FileID returns the file ID for the given channel and thread timestamp.
// FileID is the ID of the file within the directory (it's basically the
// file name without extension).  If includeThread is true and threadTS is
// not empty, the thread timestamp will be appended to the channel ID.
// Otherwise, only the channel ID will be returned.
func FileID(channelID, threadTS string, includeThread bool) string {
	if includeThread && threadTS != "" {
		return channelID + "-" + threadTS
	}
	return channelID
}

func SplitFileID(fileID string) (channelID, threadTS string) {
	channelID, threadTS, _ = strings.Cut(fileID, "-")
	return
}

// RemoveAll deletes the directory and all its contents.  Make sure all files
// are closed.
func (d *Directory) RemoveAll() error {
	return os.RemoveAll(d.dir)
}

var errNoChannelInfo = errors.New("no channel info")

// Channels collects all channels from the chunk directory.  First, it
// attempts to find the channel.json.gz file, if it's not present, it will go
// through all conversation files and try to get "ChannelInfo" chunk from the
// each file.
func (d *Directory) Channels() ([]slack.Channel, error) {
	// try to open the channels file
	if fi, err := os.Stat(d.filename(FChannels)); err == nil && !fi.IsDir() {
		return loadChannelsJSON(d.filename(FChannels))
	}
	// channel files not found, try to get channel info from the conversation
	// files.
	var ch []slack.Channel
	if err := filepath.WalkDir(d.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ext) {
			return nil
		} else if d.IsDir() {
			return nil
		}
		chs, err := loadChanInfo(path)
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
	return ch, nil
}

func loadChanInfo(fullpath string) ([]slack.Channel, error) {
	f, err := openChunks(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readChanInfo(f)
}

// readChanInfo returns the Channels from all the ChannelInfo chunks in the
// file.
func readChanInfo(rs io.ReadSeeker) ([]slack.Channel, error) {
	cf, err := FromReader(rs)
	if err != nil {
		return nil, err
	}
	return cf.AllChannelInfos()
}

// loadChannelsJSON loads channels json file and returns a slice of
// slack.Channel.  It expects it to be GZIP compressed.
func loadChannelsJSON(fullpath string) ([]slack.Channel, error) {
	f, err := openChunks(fullpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cf, err := FromReader(f)
	if err != nil {
		return nil, err
	}
	return cf.AllChannels()
}

// openChunks opens an existing chunk file and returns a ReadSeekCloser.  It
// expects a chunkfile to be a gzip-compressed file.
func openChunks(filename string) (io.ReadSeekCloser, error) {
	if fi, err := os.Stat(filename); err != nil {
		return nil, err
	} else if fi.IsDir() {
		return nil, errors.New("chunk file is a directory")
	} else if fi.Size() == 0 {
		return nil, errors.New("chunk file is empty")
	}
	f, err := os.Open(filename)
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
func (d *Directory) Open(fileID string) (*File, error) {
	f, err := d.OpenRAW(d.filename(fileID))
	if err != nil {
		return nil, err
	}
	return FromReader(f)
}

// OpenRAW opens a compressed chunk file with filename within the directory,
// and returns a ReadSeekCloser.  filename is the full name of the file with
// extension.
func (d *Directory) OpenRAW(filename string) (io.ReadSeekCloser, error) {
	return openChunks(filepath.Join(filename))
}

// filename returns the full path of the chunk file with the given fileID.
func (d *Directory) filename(fileID string) string {
	return filepath.Join(d.dir, fileID+ext)
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
func (d *Directory) Create(fileID string) (io.WriteCloser, error) {
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
	for _, name := range []string{FWorkspace, FUsers, FChannels} {
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
