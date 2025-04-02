package dbase

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository/mock_repository"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

// testDB returns a test database with the schema applied.
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.TestDB(t)
	if err := initDB(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := repository.Migrate(context.Background(), db.DB, true); err != nil {
		t.Fatal(err)
	}
	return db
}

func testPersistentDB(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.TestPersistentDB(t)
	if err := initDB(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := repository.Migrate(context.Background(), db.DB, true); err != nil {
		t.Fatal(err)
	}
	return db
}

func testDBDSN(t *testing.T, dsn string) *sqlx.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.TestDBDSN(t, dsn)
	if err := initDB(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := repository.Migrate(context.Background(), db.DB, true); err != nil {
		t.Fatal(err)
	}
	return db
}

func Test_initDB(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn *sqlx.DB
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"ok",
			args{
				ctx:  context.Background(),
				conn: testutil.TestDB(t),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := initDB(tt.args.ctx, tt.args.conn); (err != nil) != tt.wantErr {
				t.Errorf("initDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	sharedDB := testutil.TestDB(t)
	type args struct {
		ctx  context.Context
		conn *sqlx.DB
		p    SessionInfo
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *DBP
		wantErr bool
	}{
		{
			name: "initialises the database and returns the processor",
			args: args{
				ctx:  context.Background(),
				conn: sharedDB,
				p:    SessionInfo{},
			},
			want: &DBP{
				conn:      sharedDB,
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.conn, tt.args.p, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_Close(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
	}
	tests := []struct {
		name    string
		fields  fields
		prepFn  utilityFunc
		checkFn func(t *testing.T, conn sqlx.QueryerContext)
		wantErr bool
	}{
		{
			name: "finalises existing session",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			prepFn: prepSession,
			checkFn: func(t *testing.T, conn sqlx.QueryerContext) {
				var count int
				if err := conn.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM session WHERE id = 1 and finished = true").Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Errorf("session not finalised")
				}
			},
			wantErr: false,
		},
		{
			name: "session not found",
			fields: fields{
				conn:      testDB(t),
				sessionID: 2,
			},
			prepFn:  prepSession,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
			}
			if err := d.Close(); (err != nil) != tt.wantErr {
				t.Errorf("DBP.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.fields.conn)
			}
		})
	}
	// special cases not covered by the above tests
	t.Run("closed", func(t *testing.T) {
		d := &DBP{
			conn:      testDB(t),
			sessionID: 1,
		}
		sr := repository.NewSessionRepository()
		_, err := sr.Insert(context.Background(), d.conn, &repository.Session{})
		if err != nil {
			t.Fatal(err)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("no session", func(t *testing.T) {
		d := &DBP{
			conn:      testDB(t),
			sessionID: 1,
		}
		if err := d.Close(); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("no session table", func(t *testing.T) {
		d := &DBP{
			conn:      testutil.TestDB(t),
			sessionID: 1,
		}
		if err := d.Close(); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDBP_String(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"String",
			fields{
				sessionID: 42,
			},
			"<DBP:42>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
			}
			if got := d.String(); got != tt.want {
				t.Errorf("DBP.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_IsComplete(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectfn func(mmr *mock_repository.MockMessageRepository)
		want     bool
		wantErr  bool
	}{
		{
			name: "is complete",
			fields: fields{
				conn:      testDB(t),
				sessionID: 42,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C123456",
			},
			expectfn: func(mmr *mock_repository.MockMessageRepository) {
				mmr.EXPECT().CountUnfinished(gomock.Any(), gomock.Any(), int64(42), "C123456").Return(int64(0), nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "is not complete",
			fields: fields{
				conn:      testDB(t),
				sessionID: 42,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C123456",
			},
			expectfn: func(mmr *mock_repository.MockMessageRepository) {
				mmr.EXPECT().CountUnfinished(gomock.Any(), gomock.Any(), int64(42), "C123456").Return(int64(1), nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 42,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C123456",
			},
			expectfn: func(mmr *mock_repository.MockMessageRepository) {
				mmr.EXPECT().CountUnfinished(gomock.Any(), gomock.Any(), int64(42), "C123456").Return(int64(0), assert.AnError)
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "no rows",
			fields: fields{
				conn:      testDB(t),
				sessionID: 42,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C123456",
			},
			expectfn: func(mmr *mock_repository.MockMessageRepository) {
				mmr.EXPECT().CountUnfinished(gomock.Any(), gomock.Any(), int64(42), "C123456").Return(int64(0), sql.ErrNoRows)
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mmr := mock_repository.NewMockMessageRepository(ctrl)
			if tt.expectfn != nil {
				tt.expectfn(mmr)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        mmr,
			}
			got, err := d.IsComplete(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.IsComplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_Source(t *testing.T) {
	sharedDB := testutil.TestDB(t)
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
	}
	tests := []struct {
		name   string
		fields fields
		want   *Source
	}{
		{
			name: "creates new source",
			fields: fields{
				conn:      sharedDB,
				sessionID: 42,
			},
			want: &Source{
				conn:     sharedDB,
				canClose: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
			}
			if got := d.Source(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DBP.Source() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_Encode(t *testing.T) {
	type fields struct {
		conn *sqlx.DB
	}
	type args struct {
		ctx context.Context
		ch  *chunk.Chunk
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				conn: testDB(t),
			},
			args: args{
				ctx: context.Background(),
				ch: &chunk.Chunk{
					Type:      chunk.CMessages,
					Timestamp: time.Now().UnixNano(),
					ChannelID: "C123",
					Count:     1,
					IsLast:    true,
					Messages:  []slack.Message{{}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid chunk type",
			fields: fields{
				conn: testDB(t),
			},
			args: args{
				ctx: context.Background(),
				ch: &chunk.Chunk{
					Type:      0xCC,
					Timestamp: time.Now().UnixNano(),
					ChannelID: "C123",
					Count:     1,
					IsLast:    true,
					Messages:  []slack.Message{{}},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(tt.args.ctx, tt.fields.conn, SessionInfo{})
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Encode(tt.args.ctx, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("DBP.Encode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
