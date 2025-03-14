package dbase

import (
	"context"
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"

	"github.com/jmoiron/sqlx"

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
}
