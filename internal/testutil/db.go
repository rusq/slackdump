package testutil

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const Driver = "sqlite"

func TestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	return TestDBDSN(t, ":memory:")
}

func TestDBDSN(t *testing.T, dsn string) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open(Driver, dsn)
	if err != nil {
		t.Fatalf("TestDBDSN: %s: %s", dsn, err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("TestDBDSN: %s: %s", dsn, err)
	}
	return db
}

func TestPersistentDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dir := t.TempDir()
	// name is the hash of the test name
	namehash := sha1.Sum([]byte(t.Name()))
	name := hex.EncodeToString(namehash[:4])
	dbfile := filepath.Join(dir, name+".db")
	db := TestDBDSN(t, filepath.Join(dir, name))
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("TestPersistentDB: %s", dbfile)
		} else {
			t.Logf("TestPersistentDB: %s", dbfile)
			if err := os.Remove(dbfile); err != nil {
				t.Logf("TestPersistentDB: %s: %s", dbfile, err)
			}
		}
	})
	return db
}
