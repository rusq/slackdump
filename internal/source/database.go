package source

import (
	"context"
	"iter"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
)

type Database struct {
	name    string
	s       *dbproc.Source
	files   Storage
	avatars Storage
}

// OpenDatabase attempts to open the database at given path. It supports both
// types - when database file is given directly, and when the path is a
// directory containing the "slackdump.sqlite" file.  In the latter case, it
// will also attempt to open the mattermost storage.
func OpenDatabase(path string) (*Database, error) {
	var (
		fst    Storage = fstNotFound{}
		ast    Storage = fstNotFound{}
		dbfile string
		name   string
	)

	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else if !fi.IsDir() {
		// direct file
		dbfile = path
		name = filepath.Dir(path)
	} else {
		// directory
		dbfile = filepath.Join(path, "slackdump.sqlite")
		rootFS := os.DirFS(path)
		if st, err := NewMattermostStorage(rootFS); err == nil {
			fst = st
		}
		if st, err := NewAvatarStorage(os.DirFS(path)); err == nil {
			ast = st
		}
		name = path
	}

	s, err := dbproc.Open(dbfile)
	if err != nil {
		return nil, err
	}

	return &Database{name: name, s: s, files: fst, avatars: ast}, nil
}

// OpenDatabaseConn uses existing connection to the database.  It does not
// attempt to open storage etc.
func OpenDatabaseConn(conn *sqlx.DB) *Database {
	return &Database{name: "unknown", s: dbproc.Connect(conn), files: fstNotFound{}, avatars: fstNotFound{}}
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

func (d *Database) AllMessages(ctx context.Context, channelID string) (iter.Seq2[slack.Message, error], error) {
	return d.s.AllMessages(ctx, channelID)
}

func (d *Database) AllThreadMessages(ctx context.Context, channelID, threadID string) (iter.Seq2[slack.Message, error], error) {
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

func (d *Database) Files() Storage {
	return d.files
}

func (d *Database) Avatars() Storage {
	return d.avatars
}

func (d *Database) Sorted(ctx context.Context, channelID string, desc bool, cb func(ts time.Time, msg *slack.Message) error) error {
	return d.s.Sorted(ctx, channelID, desc, cb)
}
