package repository

import (
	"context"
	"iter"
	"log/slog"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

const (
	sqliteShared = "file::memory:?cache=shared"
	sqliteMemory = ":memory:"
)

var TEST_DEBUG = os.Getenv("TEST_DEBUG") == "1"

// testConn returns a new in-memory database connection for testing.
func testConn(t *testing.T) *sqlx.DB {
	t.Helper()
	if TEST_DEBUG {
		return testConnDSN(t, t.Name()+".sqlite")
	}
	return testConnDSN(t, sqliteMemory)
}

func testConnDSN(t *testing.T, dsn string) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open(dbDriver, dsn)
	if err != nil {
		t.Fatalf("sql.Open() err = %v; want nil", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() err = %v; want nil", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := Migrate(context.Background(), db.DB); err != nil {
		t.Fatalf("Migrate() err = %v; want nil", err)
	}
	return db
}

// utilityFn is a utility function to prepare the database for the test or
// check results.
type utilityFn func(t *testing.T, conn sqlx.ExtContext)

// checkCount returns a utility function to check the count of rows in the table.
func checkCount(table string, want int) utilityFn {
	return func(t *testing.T, conn sqlx.ExtContext) {
		t.Helper()
		var count int
		if err := conn.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatalf(" err = %v; want nil", err)
		}
		if count != want {
			t.Errorf("count = %d; want %d", count, want)
		}
	}
}

// ptr returns a pointer to the value.
func ptr[T any](v T) *T {
	return &v
}

// testResult is a pair of value and error to use in the test iterators.
type testResult[T any] struct {
	V   T
	Err error
}

func toTestResult[T dbObject](v T, err error) testResult[T] {
	return testResult[T]{V: v, Err: err}
}

// toIter converts a slice of testResult to an iter.Seq2.
func toIter[T any](v []testResult[T]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, r := range v {
			if !yield(r.V, r.Err) {
				break
			}
		}
	}
}
