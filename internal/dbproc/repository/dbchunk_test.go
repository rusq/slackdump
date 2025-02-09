package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/stretchr/testify/assert"
)

func Test_chunkRepository_Insert(t *testing.T) {
	type args struct {
		ctx   context.Context
		conn  sqlx.ExtContext
		chunk *DBChunk
	}
	tests := []struct {
		name    string
		args    args
		prepFn  utilityFn
		want    int64
		wantErr assert.ErrorAssertionFunc
		checkFn utilityFn
	}{
		{
			name: "success",
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
				chunk: &DBChunk{
					SessionID:  1,
					UnixTS:     1234567890,
					TypeID:     chunk.CFiles,
					NumRecords: 100,
					Final:      true,
				},
			},
			prepFn: func(t *testing.T, db sqlx.ExtContext) {
				var r sessionRepository
				id, err := r.Insert(context.Background(), db, &Session{})
				require.NoError(t, err)
				assert.Equal(t, int64(1), id)
			},
			want:    1,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			c := chunkRepository{}
			got, err := c.Insert(tt.args.ctx, tt.args.conn, tt.args.chunk)
			if !tt.wantErr(t, err, fmt.Sprintf("Insert(%v, %v, %v)", tt.args.ctx, tt.args.conn, tt.args.chunk)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Insert(%v, %v, %v)", tt.args.ctx, tt.args.conn, tt.args.chunk)
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.conn)
			}
		})
	}
}
