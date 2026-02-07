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
package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

const defFilename = "slackdump.sqlite"

// Database returns the database connection open for writing, and a session
// info based on the mode and the command line arguments.
func Database(dir string, mode string) (*sqlx.DB, dbase.SessionInfo, error) {
	dbfile := filepath.Join(dir, defFilename)
	db, err := sqlx.Open(repository.Driver, dbfile)
	if err != nil {
		return nil, dbase.SessionInfo{}, err
	}
	if err := db.Ping(); err != nil {
		return nil, dbase.SessionInfo{}, err
	}
	return db, SessionInfo(mode), nil
}

func SessionInfo(mode string) dbase.SessionInfo {
	var args string
	if len(os.Args) > 1 {
		args = strings.Join(os.Args[1:], "|")
	}

	si := dbase.SessionInfo{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.WithFiles,
		AvatarsEnabled: cfg.WithAvatars,
		Mode:           mode,
		Args:           args,
	}
	return si
}
