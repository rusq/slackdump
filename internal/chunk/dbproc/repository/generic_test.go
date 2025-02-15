package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"

	"github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/stretchr/testify/assert"
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
	tests := []testCase[*DBChannel]{
		{
			name: "returns most recent versions",
			r:    genericRepository[*DBChannel]{t: new(DBChannel)},
			args: args{
				ctx:    context.Background(),
				conn:   testConn(t),
				typeID: chunk.CChannelInfo,
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunk(chunk.CChannelInfo)(t, conn)
				cir := NewChannelRepository()
				_, err := cir.InsertAll(context.Background(), conn, toIter([]testResult[*DBChannel]{
					{V: &DBChannel{ID: "ABC", ChunkID: 1, Data: data1}, Err: nil},
					{V: &DBChannel{ID: "BCD", ChunkID: 1, Data: data2}, Err: nil},
				}))
				require.NoError(t, err)
			},
			want: []testResult[*DBChannel]{
				{V: &DBChannel{ID: "ABC"}, Err: nil},
				{V: &DBChannel{ID: "BCD"}, Err: nil},
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
			collected := collect(t, got)
			assert.Equal(t, tt.want, collected)
		})
	}
}
