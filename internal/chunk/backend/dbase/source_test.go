package dbase

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	type args struct {
		ctx  context.Context
		path string
	}
	tests := []struct {
		name    string
		args    args
		checkFn utilityFunc
		wantErr bool
	}{
		{
			name: "opens and migrates the database",
			args: args{
				ctx:  context.Background(),
				path: filepath.Join(dir, t.Name()+".db"),
			},
			checkFn: checkGooseTable,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Open(tt.args.ctx, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Close()
			if tt.checkFn != nil {
				tt.checkFn(t, testutil.TestDBDSN(t, tt.args.path))
			}
		})
	}
}

func TestSource_Close(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "closes the connection",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			wantErr: false,
		},
		{
			name: "does not close the connection",
			fields: fields{
				conn:     testDB(t),
				canClose: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			t.Cleanup(func() {
				if err := s.conn.Close(); err != nil {
					t.Error(err)
				}
			})
			if err := s.Close(); (err != nil) != tt.wantErr {
				t.Errorf("Source.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSource_Channels(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "returns channels",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx: context.Background(),
			},
			prepFn: func(t *testing.T, conn repository.PrepareExtContext) {
				t.Helper()
				ctx := context.Background()
				dbp, err := New(ctx, conn.(*sqlx.DB), SessionInfo{})
				if err != nil {
					t.Fatal(err)
				}
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannels)
				for _, ch := range channels {
					if err := dbp.Encode(ctx, &chunk.Chunk{Type: chunk.CChannelInfo, Channel: &ch}); err != nil {
						t.Error(err)
					}
					if len(ch.Members) > 0 {
						if err := dbp.Encode(ctx, &chunk.Chunk{Type: chunk.CChannelUsers, ChannelID: ch.ID, ChannelUsers: ch.Members}); err != nil {
							t.Error(err)
						}
					}
				}
			},
			want:    fixtures.Load[[]slack.Channel](fixtures.TestChannels),
			wantErr: false,
		},
		{
			name: "should return chunk.Channel if no ChannelInfo chunks are present",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx: context.Background(),
			},
			prepFn: func(t *testing.T, conn repository.PrepareExtContext) {
				t.Helper()
				ctx := context.Background()
				dbp, err := New(ctx, conn.(*sqlx.DB), SessionInfo{})
				if err != nil {
					t.Fatal(err)
				}
				channels := fixtures.Load[[]slack.Channel](fixtures.TestChannels)
				if err := dbp.Encode(ctx, &chunk.Chunk{Type: chunk.CChannels, Channels: channels}); err != nil {
					t.Error(err)
				}
			},
			want:    fixtures.Load[[]slack.Channel](fixtures.TestChannels),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.Channels(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.Channels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// channels are sorted by name
			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Name < tt.want[j].Name
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func checkGooseTable(t *testing.T, conn repository.PrepareExtContext) {
	t.Helper()
	var n int
	if err := conn.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM goose_db_version").Scan(&n); err != nil {
		t.Error(err)
	}
	if n == 0 {
		t.Error("database not migrated")
	}
}

func Test_migrate(t *testing.T) {
	dir := t.TempDir()
	type args struct {
		ctx  context.Context
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		checkFn utilityFunc
	}{
		{
			name: "migrates the database",
			args: args{
				ctx:  context.Background(),
				path: filepath.Join(dir, t.Name()+".db"),
			},
			wantErr: false,
			checkFn: checkGooseTable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := migrate(tt.args.ctx, tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("migrate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, testutil.TestDBDSN(t, tt.args.path))
			}
		})
	}
}

func prepTestChunk(c ...*chunk.Chunk) utilityFunc {
	return func(t *testing.T, conn repository.PrepareExtContext) {
		t.Helper()
		ctx := context.Background()
		dbp, err := New(ctx, conn.(*sqlx.DB), SessionInfo{})
		if err != nil {
			t.Fatal(err)
		}
		for _, ch := range c {
			if err := dbp.Encode(ctx, ch); err != nil {
				t.Error(err)
			}
		}
	}
}

func TestSource_channelUsers(t *testing.T) {
	testUsers := []string{"U01", "U02", "U03"}
	prepTestChannelUsers := func(ch string) func(t *testing.T, conn repository.PrepareExtContext) {
		return prepTestChunk(&chunk.Chunk{Type: chunk.CChannelUsers, ChannelID: ch, ChannelUsers: testUsers})
	}
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx       context.Context
		channelID string
		prealloc  int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []string
		wantErr bool
	}{
		{
			name: "returns channel users",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C01",
				prealloc:  10,
			},
			prepFn: prepTestChannelUsers("C01"),
			want:   testUsers,
		},
		{
			name: "returns empty slice if no users",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C02",
				prealloc:  10,
			},
			prepFn: prepTestChannelUsers("C01"),
			want:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.channelUsers(tt.args.ctx, tt.args.channelID, tt.args.prealloc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.channelUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Source.channelUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSource_Users(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []slack.User
		wantErr bool
	}{
		{
			name: "returns users",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx: context.Background(),
			},
			prepFn: prepTestChunk(&chunk.Chunk{Type: chunk.CUsers, Users: fixtures.Load[[]slack.User](fixtures.UsersJSON)}),
			want:   fixtures.Load[[]slack.User](fixtures.UsersJSON),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.Users(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.Users() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Slice(tt.want, func(i, j int) bool { // users are sorted by ID.
				return tt.want[i].ID < tt.want[j].ID
			})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Source.Users() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSource_AllMessages(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []testutil.TestResult[slack.Message]
		wantErr bool
	}{
		{
			name: "returns messages",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C01",
			},
			prepFn: prepTestChunk(&chunk.Chunk{Type: chunk.CMessages, ChannelID: "C01", Messages: fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)}),
			want:   testutil.SliceToTestResult(fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.AllMessages(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.AllMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Slice(tt.want, func(i, j int) bool { // messages are sorted by timestamp.
				return tt.want[i].V.Timestamp < tt.want[j].V.Timestamp
			})
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func TestSource_AllThreadMessages(t *testing.T) {
	threadMsg := []slack.Message{
		{Msg: slack.Msg{Timestamp: "1234567890.000001", ThreadTimestamp: "1234567890.000001"}},
		{Msg: slack.Msg{Timestamp: "1234567890.000002", ThreadTimestamp: "1234567890.000001"}},
		{Msg: slack.Msg{Timestamp: "1234567890.000003", ThreadTimestamp: "1234567890.000001"}},
	}
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx       context.Context
		channelID string
		threadID  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []testutil.TestResult[slack.Message]
		wantErr bool
	}{
		{
			name: "returns thread messages",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx:       context.Background(),
				channelID: "C01",
				threadID:  threadMsg[0].ThreadTimestamp,
			},
			prepFn: prepTestChunk(&chunk.Chunk{Type: chunk.CThreadMessages, ChannelID: "C01", Parent: &threadMsg[0], Messages: threadMsg}),
			want:   testutil.SliceToTestResult(threadMsg),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.AllThreadMessages(tt.args.ctx, tt.args.channelID, tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.AllThreadMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Slice(tt.want, func(i, j int) bool { // messages are sorted by timestamp.
				return tt.want[i].V.Timestamp < tt.want[j].V.Timestamp
			})
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func TestSource_Sorted(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx       context.Context
		channelID string
		desc      bool
		cb        func(ts time.Time, msg *slack.Message) error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			if err := s.Sorted(tt.args.ctx, tt.args.channelID, tt.args.desc, tt.args.cb); (err != nil) != tt.wantErr {
				t.Errorf("Source.Sorted() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSource_ChannelInfo(t *testing.T) {
	testChannel := slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID:         "C1234567890",
				Created:    1580000000,
				IsOpen:     false,
				NumMembers: 3,
			},
			Name:       "test-channel",
			Creator:    "",
			IsArchived: false,
			Members:    []string{"U01", "U02", "U03"},
		},
		IsChannel: true,
		IsGeneral: true,
		IsMember:  true,
	}

	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx       context.Context
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    *slack.Channel
		wantErr bool
	}{
		{
			name: "returns channel info",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx:       context.Background(),
				channelID: testChannel.ID,
			},
			prepFn: prepTestChunk(
				&chunk.Chunk{Type: chunk.CChannelInfo, Channel: &testChannel},
				&chunk.Chunk{Type: chunk.CChannelUsers, ChannelID: testChannel.ID, ChannelUsers: testChannel.Members},
			),
			want: &testChannel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.ChannelInfo(tt.args.ctx, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.ChannelInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Source.ChannelInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSource_WorkspaceInfo(t *testing.T) {
	testAuthTest := &slack.AuthTestResponse{
		URL:          "https://test.slack.com/",
		Team:         "Test Team",
		User:         "Test User",
		TeamID:       "T1234567890",
		UserID:       "U1234567890",
		EnterpriseID: "E1234567890",
		BotID:        "B1234567890",
	}
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    *slack.AuthTestResponse
		wantErr bool
	}{
		{
			name: "returns workspace info",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx: context.Background(),
			},
			prepFn: prepTestChunk(&chunk.Chunk{Type: chunk.CWorkspaceInfo, WorkspaceInfo: testAuthTest}),
			want:   testAuthTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.WorkspaceInfo(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.WorkspaceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Source.WorkspaceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSource_Latest(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[structures.SlackLink]time.Time
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := s.Latest(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.Latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Source.Latest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSource_ToChunk(t *testing.T) {
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx    context.Context
		e      chunk.Encoder
		sessID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			if err := src.ToChunk(tt.args.ctx, tt.args.e, tt.args.sessID); (err != nil) != tt.wantErr {
				t.Errorf("Source.ToChunk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSource_Sessions(t *testing.T) {
	testCreatedAt := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	testUpdatedAt := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	sessions := []repository.Session{
		{ID: 1, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt, Mode: "test"},
		{ID: 2, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt, Mode: "test"},
		{ID: 3, CreatedAt: testCreatedAt, UpdatedAt: testUpdatedAt, Mode: "test"},
	}
	type fields struct {
		conn     *sqlx.DB
		canClose bool
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    []repository.Session
		wantErr bool
	}{
		{
			name: "returns sessions",
			fields: fields{
				conn:     testDB(t),
				canClose: true,
			},
			args: args{
				ctx: context.Background(),
			},
			prepFn: func(t *testing.T, ec repository.PrepareExtContext) {
				sr := repository.NewSessionRepository()
				for _, s := range sessions {
					if _, err := sr.Insert(context.Background(), ec, &s); err != nil {
						t.Error(err)
					}
				}
			},
			want:    sessions,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.fields.conn)
			}
			src := &Source{
				conn:     tt.fields.conn,
				canClose: tt.fields.canClose,
			}
			got, err := src.Sessions(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.Sessions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
