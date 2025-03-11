package source

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
)

type Database struct {
	name    string
	files   Storage
	avatars Storage
	*dbproc.Source
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
		if st, err := NewMattermostStorage(rootFS); err == nil {
			fst = st
		}
		if st, err := NewAvatarStorage(os.DirFS(path)); err == nil {
			ast = st
		}
		name = path
	}

	s, err := dbproc.Open(ctx, dbfile)
	if err != nil {
		return nil, err
	}

	return &Database{name: name, Source: s, files: fst, avatars: ast}, nil
}

// DatabaseWithSource returns a new database source with the given database
// processor source.
func DatabaseWithSource(source *dbproc.Source) *Database {
	return &Database{name: "dbproc", Source: source, files: fstNotFound{}, avatars: fstNotFound{}}
}

func (d *Database) SetFiles(fst Storage) *Database {
	d.files = fst
	return d
}

func (d *Database) SetAvatars(fst Storage) *Database {
	d.avatars = fst
	return d
}

func (d *Database) Name() string {
	return d.name
}

func (d *Database) Type() string {
	return "data base"
}

func (d *Database) Files() Storage {
	return d.files
}

func (d *Database) Avatars() Storage {
	return d.avatars
}
