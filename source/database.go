// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package source

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
)

const DefaultDBFile = "slackdump.sqlite"

// Database represents a database source.  It implements the [Sourcer]
// interface and provides access to the database data.  It also provides
// access to the files and avatars storage, if available.  The database source
// is created by calling [OpenDatabase] function.
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
// will also attempt to open the mattermost storage, and if no storage is found,
// it will return a special [NoStorage] type, which returns [fs.ErrNotExist] for
// all file operations.
func OpenDatabase(ctx context.Context, path string) (*Database, error) {
	var (
		fst    Storage = NoStorage{}
		ast    Storage = NoStorage{}
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
		dbfile = filepath.Join(path, DefaultDBFile)
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
// processor source.  It will not have any files or avatars storage.  In most
// cases you should use [OpenDatabase] instead, unless you know what you are
// doing.
func DatabaseWithSource(source *dbase.Source) *Database {
	return &Database{name: "dbase", Source: source, files: NoStorage{}, avatars: NoStorage{}}
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(chns) == 0 {
		return nil, ErrNotFound
	}
	return chns, nil
}

func (d *Database) WorkspaceInfo(ctx context.Context) (*slack.AuthTestResponse, error) {
	info, err := d.Source.WorkspaceInfo(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return info, nil
}
