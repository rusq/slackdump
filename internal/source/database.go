package source

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
)

type Database struct {
	name string
	s    *dbproc.Source
	Storage
}

// OpenDatabase attempts to open the database at given path. It supports both
// types - when database file is given directly, and when the path is a
// directory containing the "slackdump.sqlite" file.  In the latter case, it
// will also attempt to open the mattermost storage.
func OpenDatabase(path string) (*Database, error) {
	var (
		fst    Storage = fstNotFound{}
		dbfile string
	)

	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		// direct file
		dbfile = path
	} else {
		// directory
		dbfile = filepath.Join(path, "slackdump.sqlite")
		if st, err := NewMattermostStorage(os.DirFS(path)); err == nil {
			fst = st
		}
	}

	s, err := dbproc.Open(dbfile)
	if err != nil {
		return nil, err
	}

	return &Database{s: s, Storage: fst}, nil
}

func (d *Database) Close() error {
	return d.s.Close()
}

func (d *Database) Name() string {
	return d.name
}

func (d *Database) Type() string {
	return "data base"
}

func (d *Database) Channels(ctx context.Context) ([]slack.Channel, error) {
	return d.s.Channels(ctx)
}

func (d *Database) Users(ctx context.Context) ([]slack.User, error) {
	return d.s.Users(ctx)
}

func (d *Database) AllMessages(ctx context.Context, channelID string) ([]slack.Message, error) {
	return d.s.AllMessages(ctx, channelID)
}

func (d *Database) AllThreadMessages(ctx context.Context, channelID, threadID string) ([]slack.Message, error) {
	return d.s.AllThreadMessages(ctx, channelID, threadID)
}

func (d *Database) ChannelInfo(ctx context.Context, channelID string) (*slack.Channel, error) {
	return d.s.ChannelInfo(ctx, channelID)
}

func (d *Database) WorkspaceInfo(ctx context.Context) (*slack.AuthTestResponse, error) {
	return d.s.WorkspaceInfo(ctx)
}

func (d *Database) Latest(ctx context.Context) (map[structures.SlackLink]time.Time, error) {
	return nil, nil
	// return d.s.Latest(ctx)
}
