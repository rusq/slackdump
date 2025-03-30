package testutil

import (
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
