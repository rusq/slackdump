package source

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
)

type Database struct {
	name    string
	files   Storage
	avatars Storage
	*dbase.Source
}

var _ Sourcer = (*Database)(nil)

// OpenDatabase attempts to open the database at given path. It supports both
// types - when database file is given directly, and when the path is a
// directory containing the "slackdump.sqlite" file.  In the latter case, it
// will also attempt to open the mattermost storage.
func OpenDatabase(ctx context.Context, path string) (*Database, error) {
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
		if st, err := OpenMattermostStorage(rootFS); err == nil {
			fst = st
		}
		if st, err := NewAvatarStorage(os.DirFS(path)); err == nil {
			ast = st
		}
		name = path
	}

	s, err := dbase.Open(ctx, dbfile)
	if err != nil {
		return nil, err
	}

	return &Database{name: name, Source: s, files: fst, avatars: ast}, nil
}

// DatabaseWithSource returns a new database source with the given database
// processor source.  It will not have any files or avatars storage.
func DatabaseWithSource(source *dbase.Source) *Database {
	return &Database{name: "dbase", Source: source, files: fstNotFound{}, avatars: fstNotFound{}}
}

// SetFiles sets the files storage.
func (d *Database) SetFiles(fst Storage) *Database {
	d.files = fst
	return d
}

// SetAvatars sets the avatars storage.
func (d *Database) SetAvatars(fst Storage) *Database {
	d.avatars = fst
	return d
}

func (d *Database) Name() string {
	return d.name
}

func (d *Database) Type() Flags {
	return FDatabase
}

func (d *Database) Files() Storage {
	return d.files
}

func (d *Database) Avatars() Storage {
	return d.avatars
}

func (d *Database) Channels(ctx context.Context) ([]slack.Channel, error) {
	chns, err := d.Source.Channels(ctx)
	if err != nil {
		return nil, err
	}
	if len(chns) == 0 {
		return nil, ErrNotFound
	}
	return chns, nil
}
