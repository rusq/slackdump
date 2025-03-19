package repository

import (
	"context"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
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
	if err := Migrate(context.Background(), db.DB, true); err != nil {
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
			chunkID, err := cr.Insert(ctx, conn, &c)
			if err != nil {
				t.Fatalf("chunk insert: %v", err)
			}
			t.Logf("chunk id: %d type: %s", chunkID, c.TypeID)
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

var deflatedMsgA = []byte{0xaa, 0x86, 0x98, 0x64, 0xa5, 0xe4, 0xa8, 0xa4, 0xa3, 0x54, 0x52, 0xac, 0x64, 0xa5, 0x64, 0x68, 0x64, 0xac, 0x67, 0x62, 0x6a, 0xa6, 0xa4, 0x83, 0xe9, 0x5e, 0x2b, 0xb0, 0xd7, 0x75, 0x30, 0x42, 0x10, 0x26, 0xe, 0xf7, 0xba, 0x55, 0x35, 0x72, 0xd8, 0x59, 0x29, 0x29, 0xe9, 0xa0, 0x5, 0xad, 0x15, 0xd8, 0x11, 0x3a, 0xb0, 0x98, 0x82, 0x70, 0x1, 0x1, 0x0, 0x0, 0xff, 0xff}

func Test_marshalflate(t *testing.T) {
	type args struct {
		a any
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "marshals data",
			args: args{a: msgA},
			want: deflatedMsgA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalflate(tt.args.a)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalflate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("marshalflate() = %#v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unmarshalflate(t *testing.T) {
	type args struct {
		data []byte
		v    any
	}
	tests := []struct {
		name    string
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "decompresses data",
			args: args{data: deflatedMsgA, v: new(slack.Message)},
			want: &msgA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshalflate(tt.args.data, tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("unmarshalflate() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.args.v, tt.want)
		})
	}
}
