package repository

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSN(t *testing.T) {
	const pragmas = "_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	tests := []struct {
		name string
		path string
		mode string
		want string
	}{
		{"default mode", "/tmp/a/slackdump.sqlite", ModeDefault, "file:/tmp/a/slackdump.sqlite?" + pragmas},
		{"read-only", "db.sqlite", ModeRO, "file:db.sqlite?mode=ro&" + pragmas},
		{"read-write", "db.sqlite", ModeRW, "file:db.sqlite?mode=rw&" + pragmas},
		{"special characters", "/x/a b?c#d%e.sqlite", ModeDefault, "file:/x/a b%3fc%23d%25e.sqlite?" + pragmas},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DSN(tt.path, tt.mode))
		})
	}

	t.Run("opens file with special characters and enables foreign keys on all connections", func(t *testing.T) {
		name := "a b?c#d%e.sqlite"
		if runtime.GOOS == "windows" {
			name = "a b#c%d.sqlite" // '?' is not allowed in Windows file names
		}
		path := filepath.Join(t.TempDir(), name)
		db, err := sqlx.Open(Driver, DSN(path, ModeDefault))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		_, err = db.Exec("CREATE TABLE T (ID INTEGER)")
		require.NoError(t, err)

		_, err = os.Stat(path)
		require.NoError(t, err, "database file must be created at the unescaped path")

		AssertForeignKeysOnAllConns(t, db, 3)
	})
}

// AssertForeignKeysOnAllConns checks that foreign keys are enabled on n
// distinct pooled connections of db.
func AssertForeignKeysOnAllConns(t *testing.T, db *sqlx.DB, n int) {
	t.Helper()
	ctx := context.Background()
	db.SetMaxOpenConns(n)
	conns := make([]*sqlx.Conn, 0, n)
	t.Cleanup(func() {
		for _, c := range conns {
			c.Close()
		}
	})
	for i := range n {
		c, err := db.Connx(ctx)
		require.NoError(t, err)
		conns = append(conns, c)
		var fk int
		require.NoError(t, c.QueryRowxContext(ctx, "PRAGMA foreign_keys").Scan(&fk))
		assert.Equal(t, 1, fk, "foreign_keys on connection %d", i)
	}
}
