package repository

import (
	"testing"

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
