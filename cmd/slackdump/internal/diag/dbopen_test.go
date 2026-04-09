package diag

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

func TestEnsureDb(t *testing.T) {
	ctx := context.Background()

	t.Run("opens archive directory", func(t *testing.T) {
		archiveDir := newArchiveDir(t)

		conn, err := ensureDb(ctx, archiveDir)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})
	})

	t.Run("rejects direct sqlite path", func(t *testing.T) {
		archiveDir := newArchiveDir(t)
		dbFile := filepath.Join(archiveDir, "slackdump.sqlite")

		conn, err := ensureDb(ctx, dbFile)
		require.Nil(t, conn)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error opening database")
	})
}

func newArchiveDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	dbFile := filepath.Join(dir, "slackdump.sqlite")

	conn, err := sql.Open(repository.Driver, dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	require.NoError(t, repository.Migrate(context.Background(), conn, false))

	return dir
}
