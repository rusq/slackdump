package dbproc

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
)

type utilityFunc func(t *testing.T, ec repository.PrepareExtContext)

var testChunk = &chunk.Chunk{
	Timestamp:   1,
	Type:        chunk.CMessages,
	Count:       1,
	ChannelID:   "1",
	SearchQuery: "1",
	IsLast:      true,
}

func prepSession(t *testing.T, ec repository.PrepareExtContext) {
	t.Helper()
	sr := repository.NewSessionRepository()
	if id, err := sr.Insert(context.Background(), ec, &repository.Session{
		ID: 1,
	}); err != nil {
		t.Fatal(err)
	} else if id != 1 {
		t.Fatalf("Insert session: want 1, got %d", id)
	}
}

func TestDBP_UnsafeInsertChunk(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
	}
	type args struct {
		ctx context.Context
		txx repository.PrepareExtContext
		ch  *chunk.Chunk
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int64
		wantErr bool
	}{
		{
			name: "inserts chunk",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx: context.Background(),
				txx: testDB(t),
				ch:  testChunk,
			},
			prepFn:  prepSession,
			want:    1,
			wantErr: false,
		},
		{
			name: "no session returns an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx: context.Background(),
				txx: testDB(t),
				ch:  testChunk,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.txx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
			}
			got, err := d.UnsafeInsertChunk(tt.args.ctx, tt.args.txx, tt.args.ch)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.UnsafeInsertChunk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.UnsafeInsertChunk() = %v, want %v", got, tt.want)
			}
		})
	}
}
