package repository

import (
	"context"
	"iter"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk"

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
type utilityFn func(t *testing.T, conn PrepareExtContext)

// checkCount returns a utility function to check the count of rows in the table.
func checkCount(table string, want int) utilityFn {
	return func(t *testing.T, conn PrepareExtContext) {
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

// prepChunk prepares number of chunks in the database.
func prepChunk(typeID ...chunk.ChunkType) utilityFn {
	return func(t *testing.T, conn PrepareExtContext) {
		t.Helper()
		ctx := context.Background()
		var (
			sr = NewSessionRepository()
			cr = NewChunkRepository()
		)
		id, err := sr.Insert(ctx, conn, &Session{ID: 1})
		if err != nil {
			t.Fatalf("session insert: %v", err)
		}
		t.Log("session id", id)
		for i, tid := range typeID {
			c := DBChunk{ID: int64(i + 1), SessionID: id, UnixTS: time.Now().UnixMilli(), TypeID: tid}
			id, err = cr.Insert(ctx, conn, &c)
			if err != nil {
				t.Fatalf("chunk insert: %v", err)
			}
			t.Log("chunk id", id)
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

func collect[T any](t *testing.T, it iter.Seq2[T, error]) []testResult[T] {
	t.Helper()
	var ret []testResult[T]
	for v, err := range it {
		ret = append(ret, testResult[T]{v, err})
	}
	return ret
}
