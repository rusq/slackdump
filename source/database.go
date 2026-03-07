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

// dbOpenParams holds the resolved paths and storages for opening a database.
type dbOpenParams struct {
	dbfile  string
	name    string
	files   Storage
	avatars Storage
}

// resolveDBPath resolves the database file, name, and optional storages for
// the given path, which may be either a direct database file or a directory.
func resolveDBPath(path string) (dbOpenParams, error) {
	p := dbOpenParams{files: NoStorage{}, avatars: NoStorage{}}
	fi, err := os.Stat(path)
	if err != nil {
		return p, err
	}
	if !fi.IsDir() {
		p.dbfile = path
		p.name = filepath.Dir(path)
	} else {
		p.dbfile = filepath.Join(path, DefaultDBFile)
		rootFS := os.DirFS(path)
		if st, err := OpenMattermostStorage(rootFS); err == nil {
			p.files = st
		}
		if st, err := NewAvatarStorage(os.DirFS(path)); err == nil {
			p.avatars = st
		}
		p.name = path
	}
	return p, nil
}

// OpenDatabase attempts to open the database at given path for reading.
// It supports both types - when database file is given directly, and when
// the path is a directory containing the "slackdump.sqlite" file.  In the
// latter case, it will also attempt to open the mattermost storage, and if
// no storage is found, it will return a special [NoStorage] type, which
// returns [fs.ErrNotExist] for all file operations.
//
// The returned [Database] does not support alias editing.  Use
// [OpenDatabaseRW] when alias write capability is needed (e.g. the viewer).
func OpenDatabase(ctx context.Context, path string) (*Database, error) {
	p, err := resolveDBPath(path)
	if err != nil {
		return nil, err
	}
	s, err := dbase.Open(ctx, p.dbfile)
	if err != nil {
		return nil, err
	}
	return &Database{name: p.name, Source: s, files: p.files, avatars: p.avatars}, nil
}

// RWDatabase is a [Database] that also supports alias write operations.
// It satisfies the viewer Aliaser interface and is returned by
// [OpenDatabaseRW] when the database file is writable.
type RWDatabase struct {
	*Database
	rw *dbase.RWSource
}

func (d *RWDatabase) SetAlias(id, alias string) error { return d.rw.SetAlias(id, alias) }
func (d *RWDatabase) DeleteAlias(id string) error     { return d.rw.DeleteAlias(id) }

// openRWFn and openFn are the functions used by OpenDatabaseRW to open the
// underlying database.  They are package-level variables so that tests can
// replace them to simulate failure scenarios without requiring filesystem
// tricks (e.g. read-only mounts) that would also block the fallback path.
var (
	openRWFn = dbase.OpenRW
	openFn   = dbase.Open
)

// OpenDatabaseRW attempts to open the database at the given path for reading
// and writing, returning an [*RWDatabase] that satisfies the viewer Aliaser
// interface.  If the database file is not writable (e.g. a read-only
// filesystem or insufficient permissions), it transparently falls back to a
// read-only [*Database].
func OpenDatabaseRW(ctx context.Context, path string) (SourceResumeCloser, error) {
	p, err := resolveDBPath(path)
	if err != nil {
		return nil, err
	}
	rw, err := openRWFn(ctx, p.dbfile)
	if err != nil {
		// Cannot open rw — fall back to ro (read-only filesystem, missing
		// write permissions, etc.).
		s, err2 := openFn(ctx, p.dbfile)
		if err2 != nil {
			return nil, err2
		}
		return &Database{name: p.name, Source: s, files: p.files, avatars: p.avatars}, nil
	}
	db := &Database{name: p.name, Source: rw.Source, files: p.files, avatars: p.avatars}
	return &RWDatabase{Database: db, rw: rw}, nil
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
