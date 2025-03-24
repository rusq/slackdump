package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func Test_genericRepository_allOfTypeWhere(t *testing.T) {
	allTestChans := fixtures.Load[[]slack.Channel](fixtures.TestChannelsJSON)
	data1 := must(marshal(allTestChans[0]))
	data2 := must(marshal(allTestChans[1]))

	type args struct {
		ctx    context.Context
		conn   sqlx.QueryerContext
		qp     queryParams
		typeID []chunk.ChunkType
	}
	type testCase[T dbObject] struct {
		name    string
		r       genericRepository[T]
		args    args
		prepFn  utilityFn
		want    []testutil.TestResult[T]
		wantErr assert.ErrorAssertionFunc
	}
	tests := []testCase[DBChannel]{
		{
			name: "returns most recent versions",
			r:    genericRepository[DBChannel]{t: DBChannel{}},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: []chunk.ChunkType{chunk.CChannelInfo},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannelInfo)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, testutil.ToIter([]testutil.TestResult[*DBChannel]{
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
				}))
				require.NoError(t, err)
			},
			want: []testutil.TestResult[DBChannel]{
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
				typeID: []chunk.ChunkType{chunk.CChannelInfo},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannels)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, testutil.ToIter([]testutil.TestResult[*DBChannel]{
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil}, // second chunk has a different type.
				}))
				require.NoError(t, err)
			},
			want: []testutil.TestResult[DBChannel]{
				{V: DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
				{V: DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Additional conditions in the query parameters",
			r:    genericRepository[DBChannel]{DBChannel{}},
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
				qp: queryParams{
					Where:   "T.ID IN (?, ?)",
					Binds:   []any{"ABC", "CDE"},
					OrderBy: []string{"T.NAME DESC"}, // NOTE: descending.
				},
				typeID: []chunk.ChunkType{chunk.CChannelInfo},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannelInfo, chunk.CChannelInfo)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, testutil.ToIter([]testutil.TestResult[*DBChannel]{
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "CDE", ChunkID: 2, Name: ptr("cde channel"), Data: data1}, Err: nil},
				}))
				require.NoError(t, err)
			},
			want: []testutil.TestResult[DBChannel]{
				{V: DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
				{V: DBChannel{ID: "CDE", ChunkID: 2, Name: ptr("cde channel"), Data: data1}, Err: nil},
			},
			wantErr: assert.NoError,
		},
		{
			name: "Same, but user key ordering (ID)",
			r:    genericRepository[DBChannel]{DBChannel{}},
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
				qp: queryParams{
					Where:        "T.ID IN (?, ?)",
					Binds:        []any{"ABC", "CDE"},
					UserKeyOrder: true,
				},
				typeID: []chunk.ChunkType{chunk.CChannelInfo},
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo, chunk.CChannelInfo, chunk.CChannelInfo)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, testutil.ToIter([]testutil.TestResult[*DBChannel]{
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Name: ptr("other name"), Data: data2}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Name: ptr("old name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
					{V: &DBChannel{ID: "CDE", ChunkID: 2, Name: ptr("cde channel"), Data: data1}, Err: nil},
				}))
				require.NoError(t, err)
			},
			want: []testutil.TestResult[DBChannel]{
				// user key is ID.
				{V: DBChannel{ID: "ABC", ChunkID: 2, Name: ptr("new name"), Data: data1}, Err: nil},
				{V: DBChannel{ID: "CDE", ChunkID: 2, Name: ptr("cde channel"), Data: data1}, Err: nil},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			got, err := tt.r.allOfTypeWhere(tt.args.ctx, tt.args.conn, tt.args.qp, tt.args.typeID...)
			if !tt.wantErr(t, err, fmt.Sprintf("allOfTypeWhere(%v, %v, %v, %v)", tt.args.ctx, tt.args.conn, tt.args.typeID, tt.args.qp)) {
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
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

func Test_genericRepository_stmtLatestWhere(t *testing.T) {
	type args struct {
		qp  queryParams
		tid []chunk.ChunkType
	}
	type testCase[T dbObject] struct {
		name  string
		r     genericRepository[T]
		args  args
		want  string
		want1 []any
	}
	tests := []testCase[DBWorkspace]{
		{
			name: "generates for empty query params",
			r:    genericRepository[DBWorkspace]{DBWorkspace{}},
			args: args{
				tid: []chunk.ChunkType{chunk.CWorkspaceInfo},
				qp:  queryParams{},
			},
			want:  "SELECT T.TEAM_ID, MAX(CHUNK_ID) AS CHUNK_ID FROM WORKSPACE AS T JOIN CHUNK AS CH ON CH.ID = T.CHUNK_ID WHERE 1=1 AND CH.TYPE_ID IN (?) GROUP BY T.TEAM_ID",
			want1: []any{chunk.CWorkspaceInfo},
		},
		{
			name: "additional predicates",
			r:    genericRepository[DBWorkspace]{DBWorkspace{}},
			args: args{
				qp: queryParams{
					Where: "NAME = ?",
					Binds: []any{2},
				},
				tid: []chunk.ChunkType{chunk.CWorkspaceInfo},
			},
			want:  "SELECT T.TEAM_ID, MAX(CHUNK_ID) AS CHUNK_ID FROM WORKSPACE AS T JOIN CHUNK AS CH ON CH.ID = T.CHUNK_ID WHERE 1=1 AND CH.TYPE_ID IN (?) AND (NAME = ?) GROUP BY T.TEAM_ID",
			want1: []any{chunk.CWorkspaceInfo, 2},
		},
		{
			name: "multiple chunk types",
			r:    genericRepository[DBWorkspace]{DBWorkspace{}},
			args: args{
				qp: queryParams{
					Where: "NAME = ?",
					Binds: []any{2},
				},
				tid: []chunk.ChunkType{chunk.CWorkspaceInfo, chunk.CMessages},
			},
			want:  "SELECT T.TEAM_ID, MAX(CHUNK_ID) AS CHUNK_ID FROM WORKSPACE AS T JOIN CHUNK AS CH ON CH.ID = T.CHUNK_ID WHERE 1=1 AND CH.TYPE_ID IN (?,?) AND (NAME = ?) GROUP BY T.TEAM_ID",
			want1: []any{chunk.CWorkspaceInfo, chunk.CMessages, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.stmtLatestWhere(tt.args.qp, tt.args.tid...)
			assert.Equalf(t, tt.want, got, "stmtLatestWhere(%v, %v)", tt.args.tid, tt.args.qp)
			assert.Equalf(t, tt.want1, got1, "stmtLatestWhere(%v, %v)", tt.args.tid, tt.args.qp)
		})
	}
}
