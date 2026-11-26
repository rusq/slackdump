package diag

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	dedupecmd "github.com/rusq/slackdump/v4/cmd/slackdump/internal/diag/dedupe"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

func Test_openDedupeDB(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	db, err := sqlx.ConnectContext(ctx, repository.Driver, repository.DSN(filepath.Join(dir, "slackdump.sqlite"), repository.ModeDefault))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	const initialSchema = 20250207082949
	require.NoError(t, goose.UpToContext(ctx, db.DB, "migrations", initialSchema))

	preview, err := openDedupeDB(ctx, dir, false)
	require.NoError(t, err)
	defer preview.Close()
	_, err = dedupecmd.Run(ctx, preview, dedupecmd.Options{})
	require.NoError(t, err)
	var count int
	require.NoError(t, preview.Get(&count, "SELECT COUNT(*) FROM sqlite_schema WHERE name = 'DEDUPE_CHECKPOINT'"))
	require.Zero(t, count)
	_, err = preview.Exec("INSERT INTO SESSION (MODE) VALUES ('test')")
	require.ErrorContains(t, err, "readonly")

	execute, err := openDedupeDB(ctx, dir, true)
	require.NoError(t, err)
	defer execute.Close()
	_, err = dedupecmd.Run(ctx, execute, dedupecmd.Options{Execute: true})
	require.NoError(t, err)
	require.NoError(t, execute.Get(&count, "SELECT COUNT(*) FROM sqlite_schema WHERE name = 'DEDUPE_CHECKPOINT'"))
	require.Equal(t, 1, count)
}

func Test_runDedupe(t *testing.T) {
	// The exit status only increases. Isolate the assertion in a new process.
	const archiveEnv = "SLACKDUMP_TEST_DEDUPE_ERROR_ARCHIVE"
	if dir := os.Getenv(archiveEnv); dir != "" {
		err := runDedupe(t.Context(), cmdDedupe, []string{"-execute", dir})
		require.ErrorContains(t, err, "forced dedupe error")
		require.Equal(t, base.SApplicationError, base.ExitStatus())
		return
	}

	ctx := t.Context()
	dir := t.TempDir()
	dsn := repository.DSN(filepath.Join(dir, "slackdump.sqlite"), repository.ModeDefault)
	db, err := sqlx.ConnectContext(ctx, repository.Driver, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	require.NoError(t, repository.Migrate(ctx, db.DB, false))
	_, err = db.Exec(`
		INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive');
		INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (1, 0, 1, 0), (2, 0, 1, 0);
		INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
		    VALUES (100, 1, 'C1', '0.000100', 0, X'7B7D'), (100, 2, 'C1', '0.000100', 0, X'7B7D');
		CREATE TRIGGER fail_dedupe BEFORE DELETE ON MESSAGE
		    BEGIN SELECT RAISE(ABORT, 'forced dedupe error'); END;`)
	require.NoError(t, err)

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^Test_runDedupe$")
	cmd.Env = append(os.Environ(), archiveEnv+"="+dir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}
