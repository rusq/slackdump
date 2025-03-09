package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
)

const defFilename = "slackdump.sqlite"

// Database returns the database connection open for writing, and a session
// info based on the mode and the command line arguments.
func Database(dir string, mode string) (*sqlx.DB, dbproc.SessionInfo, error) {
	dbfile := filepath.Join(dir, defFilename)
	db, err := sqlx.Open(repository.Driver, dbfile)
	if err != nil {
		return nil, dbproc.SessionInfo{}, err
	}
	if err := db.Ping(); err != nil {
		return nil, dbproc.SessionInfo{}, err
	}
	return db, SessionInfo(mode), nil
}

func SessionInfo(mode string) dbproc.SessionInfo {
	var args string
	if len(os.Args) > 1 {
		args = strings.Join(os.Args[1:], "|")
	}

	si := dbproc.SessionInfo{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.WithFiles,
		AvatarsEnabled: cfg.WithAvatars,
		Mode:           mode,
		Args:           args,
	}
	return si
}
