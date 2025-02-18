package repository

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

func Test_genericRepository_stmtLatest(t *testing.T) {
	type args struct {
		tid chunk.ChunkType
	}
	type testCase[T dbObject] struct {
		name      string
		r         genericRepository[T]
		args      args
		wantStmt  string
		wantBinds []any
	}
	tests := []testCase[*DBChannel]{
		{
			name:      "generates for all channels",
			r:         genericRepository[*DBChannel]{t: new(DBChannel)},
			args:      args{tid: CTypeAny},
			wantStmt:  `SELECT C.ID, MAX(CHUNK_ID) AS CHUNK_ID FROM CHANNEL AS C JOIN CHUNK AS CH ON CH.ID = C.CHUNK_ID WHERE 1=1 GROUP BY C.ID`,
			wantBinds: nil,
		},
		{
			name:      "generates for concrete type",
			r:         genericRepository[*DBChannel]{t: new(DBChannel)},
			args:      args{tid: chunk.CChannelInfo},
			wantStmt:  `SELECT C.ID, MAX(CHUNK_ID) AS CHUNK_ID FROM CHANNEL AS C JOIN CHUNK AS CH ON CH.ID = C.CHUNK_ID WHERE 1=1 AND CH.TYPE_ID = ? GROUP BY C.ID`,
			wantBinds: []any{chunk.CChannelInfo},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStmt, gotBinds := tt.r.stmtLatest(tt.args.tid)
			assert.Equalf(t, tt.wantStmt, gotStmt, "stmtLatest(%v)", tt.args.tid)
			assert.Equalf(t, tt.wantBinds, gotBinds, "stmtLatest(%v)", tt.args.tid)
		})
	}
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func Test_genericRepository_AllOfType(t *testing.T) {
	allTestChans := fixtures.Load[[]slack.Channel](fixtures.TestChannels)
	data1 := must(marshal(allTestChans[0]))
	data2 := must(marshal(allTestChans[1]))

	type args struct {
		ctx    context.Context
		conn   sqlx.QueryerContext
		typeID chunk.ChunkType
	}
	type testCase[T dbObject] struct {
		name    string
		r       genericRepository[T]
		args    args
		prepFn  utilityFn
		want    []testResult[T]
		wantErr assert.ErrorAssertionFunc
	}
	tests := []testCase[DBChannel]{
		{
			name: "returns most recent versions",
			r:    genericRepository[DBChannel]{t: DBChannel{}},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: chunk.CChannelInfo,
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannelInfo)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, toIter([]testResult[*DBChannel]{
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
				}))
				require.NoError(t, err)
			},
			want: []testResult[DBChannel]{
				{V: DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
				{V: DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "different chunk types are isolated",
			r:    genericRepository[DBChannel]{t: DBChannel{}},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: chunk.CChannelInfo,
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannels)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, toIter([]testResult[*DBChannel]{
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil}, // second chunk has a different type.
				}))
				require.NoError(t, err)
			},
			want: []testResult[DBChannel]{
				{V: DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
				{V: DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			got, err := tt.r.AllOfType(tt.args.ctx, tt.args.conn, tt.args.typeID)
			if !tt.wantErr(t, err, fmt.Sprintf("AllOfType(%v, %v, %v)", tt.args.ctx, tt.args.conn, tt.args.typeID)) {
				return
			}
			assertIterResult(t, tt.want, got)
		})
	}
}

func assertIterResult[T any](t *testing.T, want []testResult[T], got iter.Seq2[T, error]) {
	t.Helper()
	var i int
	for v, err := range got {
		assert.Equalf(t, want[i].V, v, "value %d", i)
		if (err != nil) != (want[i].Err != nil) {
			t.Errorf("got error on %d %v, want %v", i, err, want[i].Err)
		}
		i++
	}
	if i != len(want) {
		t.Errorf("got %d results, want %d", i, len(want))
	}
}

func Test_colAlias(t *testing.T) {
	type args struct {
		alias string
		col   []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "generates proper string",
			args: args{alias: "C", col: []string{"ID", "Name"}},
			want: "C.ID,C.Name",
		},
		{
			name: "no alias is not a problem",
			args: args{alias: "", col: []string{"ID", "Name"}},
			want: "ID,Name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colAlias(tt.args.alias, tt.args.col...); got != tt.want {
				t.Errorf("colAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}
