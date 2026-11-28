package diag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	dedupecmd "github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/dedupe"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/source"
)

var cmdDedupe = &base.Command{
	UsageLine:  "slackdump tools dedupe [flags] <archive_directory>",
	Short:      "deduplicate overlap entities from resume runs",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Dedupe removes identical duplicate messages, users, channels, channel users,
and files created by resume look-back overlap. Use -mode message-key to
collapse message rows by channel and timestamp even when Slack-regenerated
fields differ. In both modes, dedupe keeps the copy with the highest chunk ID
from a finished session. If none of the copies came from a finished session,
it keeps the copy with the highest chunk ID overall.

By default dedupe only reports what would be removed. Pass the directory
containing slackdump.sqlite. Use -execute to remove duplicates without a preview.

Preview opens the archive read-only and does not upgrade its schema. Archives
created before v4.5.1 lack the dedupe index until a newer version upgrades
them, so preview can be very slow on large ones and prints a note. Open such
an archive once with a command that upgrades it, such as 'slackdump view',
to add the index without removing duplicates.

Large archives are processed in batches. If execution stops, run -execute
again with the same mode to continue from the last committed batch. An archive
write or a change of mode makes the next run start a fresh pass.

The same cleanup can be run automatically after a successful resume with:

	slackdump resume -dedupe <archive_directory>
`,
}

var dedupeFlags struct {
	execute bool
	mode    repository.MessageDedupeMode
}

func init() {
	cmdDedupe.Run = runDedupe
	dedupeFlags.mode = repository.MessageDedupeExact
	cmdDedupe.Flag.BoolVar(&dedupeFlags.execute, "execute", false, "remove duplicate entities without a preview")
	cmdDedupe.Flag.Var(&dedupeFlags.mode, "mode", "message dedupe mode: exact or message-key")
}

func ensureDb(ctx context.Context, dir string) (*sqlx.DB, error) {
	src, err := source.Load(ctx, dir)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, err
	}
	defer src.Close()
	if !src.Type().Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, fmt.Errorf("source type %q does not contain a database archive, use 'slackdump convert -f database' to convert it", src.Type())
	}

	conn, err := bootstrap.Database(dir)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return conn, nil

}

func openDedupeDB(ctx context.Context, path string, execute bool) (*sqlx.DB, error) {
	if execute {
		return ensureDb(ctx, path)
	}

	// source.Load migrates database archives. Preview needs only the archive
	// type and a read-only connection, even on an older schema.
	kind, err := source.Type(path)
	if err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, err
	}
	if !kind.Has(source.FDatabase) {
		base.SetExitStatus(base.SInvalidParameters)
		return nil, fmt.Errorf("source type %q does not contain a database archive, use 'slackdump convert -f database' to convert it", kind)
	}
	dsn := repository.DSN(bootstrap.DatabasePath(path), repository.ModeRO)
	conn, err := sqlx.ConnectContext(ctx, repository.Driver, dsn)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		base.SetExitStatus(base.SInitializationError)
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return conn, nil
}

// dedupeIndexName is the MESSAGE index added by migration 20261019000000.
const dedupeIndexName = "MESSAGE_DEDUPE_IDX"

// missingIndexNote is printed before a preview of an archive without the
// message dedupe index.
const missingIndexNote = `Note: this archive was created before v4.5.1 and lacks the dedupe index.
Preview does not upgrade the archive, so it may be very slow on large archives.
To add the index without removing duplicates, open the archive once with a
command that upgrades its schema, such as 'slackdump view', then run the
preview again. 'slackdump tools dedupe -execute' also adds the index, but it
removes duplicates straight away without a preview.

`

// checkDedupeIndex warns on lg and w when the database lacks the message
// dedupe index. Preview does not migrate, so archives created before v4.5.1
// lack the index and the duplicate query can be very slow. A failed check is
// logged and never blocks the preview.
func checkDedupeIndex(ctx context.Context, lg *slog.Logger, db *sqlx.DB, path string, w io.Writer) {
	ok, err := hasDedupeIndex(ctx, db)
	if err != nil {
		lg.DebugContext(ctx, "dedupe index check failed", "path", path, "error", err)
		return
	}
	if ok {
		return
	}
	lg.WarnContext(ctx, "dedupe index missing, preview may be very slow", "path", path, "index", dedupeIndexName)
	if w != nil {
		fmt.Fprint(w, missingIndexNote)
	}
}

// hasDedupeIndex reports whether the database has the message dedupe index.
func hasDedupeIndex(ctx context.Context, db *sqlx.DB) (bool, error) {
	var n int
	if err := db.GetContext(ctx, &n, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", dedupeIndexName); err != nil {
		return false, err
	}
	return n > 0, nil
}

func runDedupe(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	if cmd.Flag.NArg() != 1 {
		cmd.Flag.Usage()
		return nil
	}

	dir := cmd.Flag.Arg(0)
	conn, err := openDedupeDB(ctx, dir, dedupeFlags.execute)
	if err != nil {
		return err
	}
	defer conn.Close()
	if !dedupeFlags.execute {
		checkDedupeIndex(ctx, cfg.Log, conn, dir, os.Stdout)
	}

	_, err = dedupecmd.Run(ctx, conn, dedupecmd.Options{
		Execute:  dedupeFlags.execute,
		Report:   os.Stdout,
		Database: dir,
		Mode:     dedupeFlags.mode,
	})
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
	}
	return err
}
