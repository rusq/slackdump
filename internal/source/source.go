// Package source provides archive readers for different output formats.
//
// Currently, the following formats are supported:
//   - archive
//   - Slack Export
//   - dump
package source

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// Sourcer is an interface for retrieving data from different sources.
//
//go:generate mockgen -destination=mock_source/mock_source.go . Sourcer,Storage
type Sourcer interface {
	// Name should return the name of the retriever underlying media, i.e.
	// directory or archive.
	Name() string
	// Type should return the type of the retriever, i.e. "chunk" or "export".
	Type() string
	// Channels should return all channels.
	Channels(ctx context.Context) ([]slack.Channel, error)
	// Users should return all users.
	Users(ctx context.Context) ([]slack.User, error)
	// AllMessages should return all messages for the given channel id.
	AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error)
	// AllThreadMessages should return all messages for the given tuple
	// (channelID, threadID). It should return the parent channel message
	// (thread lead) as a first message.
	AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error)
	// Sorted should iterate over all (both channel and thread) messages for
	// the requested channel id.  If desc is true, it must return messages in
	// descending order (by timestamp), otherwise in ascending order.  The
	// callback function cb should be called for each message. If cb returns an
	// error, the iteration should be stopped and the error should be returned.
	Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error
	// ChannelInfo should return the channel information for the given channel
	// id.
	ChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error)
	// Files should return file [Storage].
	Files() Storage
	// Avatars should return the avatar [Storage].
	Avatars() Storage
	// WorkspaceInfo should return the workspace information, if it is available.
	// If the call is not supported, it should return ErrNotSupported.
	WorkspaceInfo(ctx context.Context) (*slack.AuthTestResponse, error)
}

type SourceCloser interface {
	Sourcer
	io.Closer
}

type Resumer interface {
	// Latest should return the latest timestamps of all channels and threads.
	Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error)
}

// Resumer is the interface that should be implemented by sources that can be
// resumed.
type SourceCloseResumer interface {
	SourceCloser
	Resumer
}

var ErrNotSupported = errors.New("feature not supported")

type Flags int16

const (
	FUnknown Flags = 0
	// container
	FDirectory Flags = 1 << iota
	FZip
	// main content
	FChunk
	FExport
	FDump
	FDatabase
	// attachments
	FAvatars
	FMattermost
)

func (f Flags) String() string {
	const flg = "________MADUXCZD"
	var buf strings.Builder
	for i := 16; i >= 0; i-- {
		if f&(1<<uint(i)) != 0 {
			buf.WriteByte(flg[16-i])
		} else {
			buf.WriteByte('.')
		}
	}
	return buf.String()
}

func (f Flags) Has(ff Flags) bool {
	return f&ff == ff
}

// type assertion
var (
	_ Sourcer = &Export{}
	_ Sourcer = &ChunkDir{}
	_ Sourcer = &Dump{}
	_ Sourcer = &Database{}
)

// Load loads the source from file src.
func Load(ctx context.Context, src string) (SourceCloseResumer, error) {
	lg := slog.With("source", src)
	st, err := Type(src)
	if err != nil {
		return nil, err
	}
	if st == FUnknown {
		return nil, fmt.Errorf("unsupported source type: %s", src)
	}
	switch {
	case st.Has(FChunk | FDirectory):
		lg.DebugContext(ctx, "loading chunk directory")
		dir, err := chunk.OpenDir(src)
		if err != nil {
			return nil, err
		}
		return OpenChunkDir(dir, true), nil
	case st.Has(FExport | FZip):
		lg.DebugContext(ctx, "loading export zip")
		f, err := zip.OpenReader(src)
		if err != nil {
			return nil, err
		}
		return OpenExport(f, src)
	case st.Has(FExport | FDirectory):
		lg.DebugContext(ctx, "loading export directory")
		return OpenExport(os.DirFS(src), src)
	case st.Has(FDump | FZip):
		lg.DebugContext(ctx, "loading dump zip")
		f, err := zip.OpenReader(src)
		if err != nil {
			return nil, err
		}
		return OpenDump(ctx, f, src)
	case st.Has(FDump | FDirectory):
		lg.DebugContext(ctx, "loading dump directory")
		return OpenDump(ctx, os.DirFS(src), src)
	case st.Has(FDatabase):
		lg.DebugContext(ctx, "loading database")
		return OpenDatabase(ctx, src)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", src)
	}
}

func Type(src string) (Flags, error) {
	fi, err := os.Stat(src)
	if err != nil {
		return FUnknown, err
	}
	return srcType(src, fi), nil
}

func srcType(src string, fi fs.FileInfo) Flags {
	var fsys fs.FS // this will be our media for accessing files
	var flags Flags

	// determine container
	if fi.IsDir() {
		fsys = os.DirFS(src)
		flags |= FDirectory
	} else if fi.Mode().IsRegular() {
		if strings.ToLower(path.Ext(src)) == ".zip" {
			f, err := zip.OpenReader(src)
			if err != nil {
				return FUnknown
			}
			defer f.Close()
			fsys = f
			flags |= FZip
		} else if ext := strings.ToLower(path.Ext(src)); ext == ".db" || ext == ".sqlite" {
			flags |= FDatabase
			return flags
		} else {
			return FUnknown
		}
	} else {
		return FUnknown
	}

	// determine content

	// attachments
	if _, err := fs.Stat(fsys, "__avatars"); err == nil {
		flags |= FAvatars
	}
	if _, err := fs.Stat(fsys, chunk.UploadsDir); err == nil {
		flags |= FMattermost
	}

	// main content
	if ff, err := fs.Glob(fsys, "[CDG]*.json"); err == nil && len(ff) > 0 {
		return flags | FDump
	}

	if _, err := fs.Stat(fsys, "workspace.json.gz"); err == nil {
		if flags&FZip != 0 {
			return FUnknown // compressed chunk directories are not supported
		}
		return flags | FChunk
	} else if files, err := fs.Glob(fsys, "*.json.gz"); err == nil && len(files) > 0 {
		if flags&FZip != 0 {
			return FUnknown
		}
		return flags | FChunk
	}
	if _, err := fs.Stat(fsys, "channels.json"); err == nil {
		return flags | FExport
	}
	if _, err := fs.Stat(fsys, "slackdump.sqlite"); err == nil {
		// directory with the database
		return flags | FDatabase
	}
	return FUnknown
}

func unmarshalOne[T any](fsys fs.FS, name string) (T, error) {
	var v T
	f, err := fsys.Open(name)
	if err != nil {
		return v, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return v, err
	}
	return v, nil
}

// unmarshal reads the JSON file from the filesystem and unmarshals it into the
// provided value.
func unmarshal[T ~[]S, S any](fsys fs.FS, name string) (T, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
