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

// Database returns the initialised database connection open for writing.
func Database(dir string, mode string) (*sqlx.DB, dbproc.SessionInfo, error) {
	dbfile := filepath.Join(dir, defFilename)
	// wconn is the writer connection
	wconn, err := sqlx.Open(repository.Driver, dbfile)
	if err != nil {
		return nil, dbproc.SessionInfo{}, err
	}
	return wconn, SessionInfo(mode), nil
}

func SessionInfo(mode string) dbproc.SessionInfo {
	si := dbproc.SessionInfo{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.DownloadFiles,
		AvatarsEnabled: cfg.DownloadAvatars,
		Mode:           mode,
		Args:           strings.Join(os.Args, "|"),
	}
	return si
}
