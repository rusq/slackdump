// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"

	"github.com/rusq/slackdump/v3/internal/chunk"
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
	db, err := sqlx.Open(Driver, dsn)
	if err != nil {
		t.Fatalf("sql.Open() err = %v; want nil", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() err = %v; want nil", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("PRAGMA foreign_keys = ON err = %v; want nil", err)
	}
	if err := Migrate(t.Context(), db.DB, true); err != nil {
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
		if err := conn.QueryRowxContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
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
		tc := []testChunk{}
		for _, tid := range typeID {
			tc = append(tc, testChunk{typeID: tid, final: false})
		}
		prepChunkWithFinal(tc...)(t, conn)
	}
}

type testChunk struct {
	typeID    chunk.ChunkType
	channelID string
	final     bool
}

func prepChunkWithFinal(tc ...testChunk) utilityFn {
	return func(t *testing.T, conn PrepareExtContext) {
		t.Helper()
		ctx := t.Context()
		var (
			sr = NewSessionRepository()
			cr = NewChunkRepository()
		)
		id, err := sr.Insert(ctx, conn, &Session{ID: 1})
		if err != nil {
			t.Fatalf("session insert: %v", err)
		}
		t.Log("session id", id)
		for i, c := range tc {
			ch := DBChunk{
				ID:        int64(i + 1),
				SessionID: id,
				UnixTS:    time.Now().UnixMilli(),
				TypeID:    c.typeID,
				ChannelID: &c.channelID,
				Final:     c.final,
			}
			chunkID, err := cr.Insert(ctx, conn, &ch)
			if err != nil {
				t.Fatalf("chunk insert: %v", err)
			}
			t.Logf("chunk id: %d type: %s final: %v", chunkID, ch.TypeID, ch.Final)
		}
	}
}

// ptr returns a pointer to the value.
func ptr[T any](v T) *T {
	return &v
}

func Test_placeholders(t *testing.T) {
	type args[T any] struct {
		v []T
	}
	tests := []struct {
		name string
		args args[int]
		want []string
	}{
		{
			name: "empty",
			args: args[int]{v: nil},
			want: []string{},
		},
		{
			name: "one",
			args: args[int]{v: []int{1}},
			want: []string{"?"},
		},
		{
			name: "three",
			args: args[int]{v: []int{1, 2, 3}},
			want: []string{"?", "?", "?"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := placeholders(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("placeholders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_orNull(t *testing.T) {
	type args[T any] struct {
		b bool
		t T
	}
	tests := []struct {
		name string
		args args[int]
		want *int
	}{
		{
			name: "null",
			args: args[int]{b: false, t: 42},
			want: nil,
		},
		{
			name: "not null",
			args: args[int]{b: true, t: 42},
			want: ptr(42),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := orNull(tt.args.b, tt.args.t); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("orNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newBindAddFn(t *testing.T) {
	t.Run("adds bind", func(t *testing.T) {
		var buf strings.Builder
		var binds []any
		fn := newBindAddFn(&buf, &binds)
		fn(true, "foo = ?", 42)
		fn(false, "bar < ?", 43)
		got := buf.String()
		assert.Equal(t, got, "foo = ?")
		assert.Equal(t, binds, []any{42})
	})
}

func TestOrder_String(t *testing.T) {
	tests := []struct {
		name string
		o    Order
		want string
	}{
		{
			name: "asc",
			o:    Asc,
			want: oAsc,
		},
		{
			name: "desc",
			o:    Desc,
			want: oDesc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.String(); got != tt.want {
				t.Errorf("Order.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeQueryerContext struct{}

func (fakeQueryerContext) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (fakeQueryerContext) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return nil, nil
}

func (fakeQueryerContext) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	return nil
}

type fakeQueryerContextWRebind struct {
	fakeQueryerContext
}

func (fakeQueryerContextWRebind) Rebind(query string) string {
	return "Rebound: " + query
}

func Test_rebind(t *testing.T) {
	type args struct {
		conn sqlx.QueryerContext
		stmt string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "rebinds",
			args: args{
				conn: fakeQueryerContext{},
				stmt: "SELECT * FROM foo WHERE bar = ?",
			},
			want: "SELECT * FROM foo WHERE bar = ?", // no-op
		},
		{
			name: "rebinds",
			args: args{
				conn: fakeQueryerContextWRebind{},
				stmt: "SELECT * FROM foo WHERE bar = ?",
			},
			want: "Rebound: SELECT * FROM foo WHERE bar = ?",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rebind(tt.args.conn, tt.args.stmt); got != tt.want {
				t.Errorf("rebind() = %v, want %v", got, tt.want)
			}
		})
	}
}
