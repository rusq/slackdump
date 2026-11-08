package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestDatabase(t *testing.T) {
	ctx := context.Background()
	t.Run("enables foreign keys on every pooled connection", func(t *testing.T) {
		dir := t.TempDir()
		db, err := Database(dir)
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })

		const n = 3
		db.SetMaxOpenConns(n)
		for i := range n {
			c, err := db.Connx(ctx) // held open, so each iteration gets a new connection
			require.NoError(t, err)
			t.Cleanup(func() { c.Close() })
			var fk int
			require.NoError(t, c.QueryRowxContext(ctx, "PRAGMA foreign_keys").Scan(&fk))
			assert.Equal(t, 1, fk, "foreign_keys on connection %d", i)
		}
		assert.FileExists(t, filepath.Join(dir, defFilename))
	})
}
