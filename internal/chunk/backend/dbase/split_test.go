package dbase

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository/mock_repository"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

type utilityFunc func(t *testing.T, ec repository.PrepareExtContext)

var sampleChunk = &chunk.Chunk{
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
	if id, err := sr.Insert(t.Context(), ec, &repository.Session{
		ID: 1,
	}); err != nil {
		t.Fatal(err)
	} else if id != 1 {
		t.Fatalf("Insert session: want 1, got %d", id)
	}
}

// prepChunk prepares number of chunks in the database.
// these are duplicated from repository tests.
func prepChunk(typeID ...chunk.ChunkType) utilityFunc {
	return func(t *testing.T, conn repository.PrepareExtContext) {
		t.Helper()
		tc := []testChunk{}
		for _, tid := range typeID {
			tc = append(tc, testChunk{typeID: tid, final: false})
		}
		prepChunkWithFinal(tc...)(t, conn)
	}
}

type testChunk struct {
	typeID    chunk.ChunkType
	channelID string
	final     bool
}

func prepChunkWithFinal(tc ...testChunk) utilityFunc {
	return func(t *testing.T, conn repository.PrepareExtContext) {
		t.Helper()
		ctx := t.Context()
		var (
			sr = repository.NewSessionRepository()
			cr = repository.NewChunkRepository()
		)
		id, err := sr.Insert(ctx, conn, &repository.Session{ID: 1, Finished: true})
		if err != nil {
			t.Fatalf("session insert: %v", err)
		}
		t.Log("session id", id)
		for i, c := range tc {
			ch := repository.DBChunk{
				ID:        int64(i + 1),
				SessionID: id,
				UnixTS:    time.Now().UnixMilli(),
				TypeID:    c.typeID,
				ChannelID: &c.channelID,
				Final:     c.final,
			}
			chunkID, err := cr.Insert(ctx, conn, &ch)
			if err != nil {
				t.Fatalf("chunk insert: %v", err)
			}
			t.Logf("chunk id: %d type: %s final: %v", chunkID, ch.TypeID, ch.Final)
		}
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
				ctx: t.Context(),
				txx: testDB(t),
				ch:  sampleChunk,
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
				ctx: t.Context(),
				txx: testDB(t),
				ch:  sampleChunk,
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

func TestDBP_insertMessages(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		channelID string
		mm        []slack.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				mm: []slack.Message{
					{Msg: slack.Msg{Timestamp: "123.456", Text: "hello"}},
					{Msg: slack.Msg{Timestamp: "123.457", Text: "world"}},
				},
			},
			prepFn:  prepChunk(chunk.CMessages),
			want:    2,
			wantErr: false,
		},
		{
			name: "empty messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				mm:        []slack.Message{},
			},
			prepFn:  prepChunk(chunk.CMessages),
			want:    0,
			wantErr: false,
		},
		{
			name: "no chunk returns an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				mm: []slack.Message{
					{Msg: slack.Msg{Timestamp: "123.456", Text: "hello"}},
				},
			},
			prepFn:  nil,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertMessages(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.channelID, tt.args.mm)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_InsertChunk(t *testing.T) {
	TestDBP_UnsafeInsertChunk(t)
}

func Test_orNil(t *testing.T) {
	type args[T any] struct {
		cond bool
		v    T
	}
	tests := []struct {
		name string
		args args[int]
		want *int
	}{
		{
			name: "returns nil",
			args: args[int]{false, 1},
			want: nil,
		},
		{
			name: "returns value",
			args: args[int]{true, 1},
			want: func() *int {
				v := 1
				return &v
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := orNil(tt.args.cond, tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("orNil() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrInvalidPayload_Error(t *testing.T) {
	type fields struct {
		Type      chunk.ChunkType
		ChannelID string
		Reason    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "returns error",
			fields: fields{
				Type:      chunk.CMessages,
				ChannelID: "C123",
				Reason:    "reason",
			},
			want: "invalid payload: Messages, channel: C123, reason: reason",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ErrInvalidPayload{
				Type:      tt.fields.Type,
				ChannelID: tt.fields.ChannelID,
				Reason:    tt.fields.Reason,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("ErrInvalidPayload.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertPayload(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		c         *chunk.Chunk
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "inserts messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CMessages,
					Timestamp: 123456,
					ChannelID: "C123",
					Count:     1,
					IsLast:    true,
					Messages:  []slack.Message{{Msg: slack.Msg{Timestamp: "123.456", Text: "hello"}}},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "inserts thread messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CThreadMessages,
					Timestamp: 123456,
					ChannelID: "C123",
					ThreadTS:  "123.456",
					Count:     1,
					IsLast:    false,
					Messages:  []slack.Message{{Msg: slack.Msg{Timestamp: "123.457", ThreadTimestamp: "123.456", Text: "world"}}},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "inserts files",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CFiles,
					Timestamp: 123456,
					ChannelID: "C123",
					Parent:    &slack.Message{Msg: slack.Msg{Timestamp: "123.456", Text: "hello"}},
					Count:     1,
					IsLast:    true,
					Files:     []slack.File{{ID: "F123", Name: "file.txt"}},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "inserts workspace info",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CWorkspaceInfo,
					Timestamp: 123456,
					WorkspaceInfo: &slack.AuthTestResponse{
						Team:         "team",
						URL:          "url",
						User:         "user",
						TeamID:       "T123",
						UserID:       "U123",
						EnterpriseID: "E123",
						BotID:        "B123",
					},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "inserts users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CUsers,
					Timestamp: 123456,
					Count:     1,
					Users:     fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON)),
				},
			},
			want:    8,
			wantErr: false,
		},
		{
			name: "inserts channels",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CChannels,
					Timestamp: 123456,
					Count:     1,
					Channels:  fixtures.Load[[]slack.Channel](string(fixtures.TestExpChannelsJSON)),
				},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "inserts channel info",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CChannelInfo,
					Timestamp: 123456,
					Channel:   &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C123"}}},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "channel info, empty channel is an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:      chunk.CChannelInfo,
					Timestamp: 123456,
					Channel:   nil,
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "inserts channel users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:         chunk.CChannelUsers,
					Timestamp:    123456,
					ChannelID:    "C123",
					Count:        3,
					ChannelUsers: []string{"U123", "U124", "U125"},
				},
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "inserts search messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:        chunk.CSearchMessages,
					Timestamp:   123456,
					SearchQuery: "hello",
					Count:       1,
					SearchMessages: []slack.SearchMessage{
						{Text: "hello", Username: "user", Timestamp: "123.456"},
					},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "inserts search files",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
				mr:        repository.NewMessageRepository(),
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				c: &chunk.Chunk{
					Type:        chunk.CSearchFiles,
					Timestamp:   123456,
					SearchQuery: "hello",
					Count:       1,
					SearchFiles: []slack.File{{ID: "F123", Name: "file.txt"}},
				},
			},
			want:    1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			prepChunk(tt.args.c.Type)(t, tt.args.tx)
			got, err := d.insertPayload(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertFiles(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		channelID string
		threadTS  string
		parMsgTS  string
		ff        []slack.File
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts files",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				threadTS:  "",
				parMsgTS:  "123.456",
				ff: []slack.File{
					{ID: "F123", Name: "file.txt", Timestamp: 123456},
					{ID: "F124", Name: "file2.txt", Timestamp: 123457},
				},
			},
			prepFn:  prepChunk(chunk.CFiles),
			want:    2,
			wantErr: false,
		},
		{
			name: "empty file slice, is not an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				threadTS:  "",
				parMsgTS:  "123.456",
				ff:        []slack.File{},
			},
			prepFn:  prepChunk(chunk.CFiles),
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertFiles(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.channelID, tt.args.threadTS, tt.args.parMsgTS, tt.args.ff)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertWorkspaceInfo(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		info      *slack.AuthTestResponse
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts workspace info",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				info: &slack.AuthTestResponse{
					Team: "team",
					URL:  "url",
				},
			},
			prepFn:  prepChunk(chunk.CWorkspaceInfo),
			want:    1,
			wantErr: false,
		},
		{
			name: "empty workspace is an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				info:      nil,
			},
			prepFn:  prepChunk(chunk.CWorkspaceInfo),
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			p := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := p.insertWorkspaceInfo(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertWorkspaceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertWorkspaceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertUsers(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		users     []slack.User
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				users:     fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON)),
			},
			prepFn:  prepChunk(chunk.CUsers),
			want:    8,
			wantErr: false,
		},
		{
			name: "empty users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				users:     []slack.User{},
			},
			prepFn:  prepChunk(chunk.CUsers),
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			p := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := p.insertUsers(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.users)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertChannels(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		channels  []slack.Channel
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts channels",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channels:  fixtures.Load[[]slack.Channel](string(fixtures.TestExpChannelsJSON)),
			},
			prepFn:  prepChunk(chunk.CChannels),
			want:    2,
			wantErr: false,
		},
		{
			name: "empty channels",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channels:  []slack.Channel{},
			},
			prepFn:  prepChunk(chunk.CChannels),
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.prepFn != nil {
			tt.prepFn(t, tt.args.tx)
		}
		t.Run(tt.name, func(t *testing.T) {
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertChannels(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.channels)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertChannelUsers(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		channelID string
		users     []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts channel users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				users:     []string{"U123", "U124", "U125"},
			},
			prepFn:  prepChunk(chunk.CChannelUsers),
			want:    3,
			wantErr: false,
		},
		{
			name: "empty channel users",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "C123",
				users:     []string{},
			},
			prepFn:  prepChunk(chunk.CChannelUsers),
			want:    0,
			wantErr: false,
		},
		{
			name: "no channel ID is an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				channelID: "",
				users:     []string{"U123", "U124", "U125"},
			},
			prepFn:  prepChunk(chunk.CChannelUsers),
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertChannelUsers(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.channelID, tt.args.users)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertChannelUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertChannelUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertSearchMessages(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		in3       string
		mm        []slack.SearchMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts search messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				mm: []slack.SearchMessage{
					{Text: "hello", Username: "user", Timestamp: "123.456"},
					{Text: "world", Username: "user", Timestamp: "123.457"},
				},
			},
			prepFn:  prepChunk(chunk.CSearchMessages),
			want:    2,
			wantErr: false,
		},
		{
			name: "no messages",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				mm:        []slack.SearchMessage{},
			},
			prepFn:  prepChunk(chunk.CSearchMessages),
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertSearchMessages(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.in3, tt.args.mm)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertSearchMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertSearchMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBP_insertSearchFiles(t *testing.T) {
	type fields struct {
		conn      *sqlx.DB
		sessionID int64
		mr        repository.MessageRepository
	}
	type args struct {
		ctx       context.Context
		tx        repository.PrepareExtContext
		dbchunkID int64
		in3       string
		ff        []slack.File
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFunc
		want    int
		wantErr bool
	}{
		{
			name: "inserts search files",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				ff: []slack.File{
					{ID: "F123", Name: "file.txt", Timestamp: 123456},
					{ID: "F124", Name: "file2.txt", Timestamp: 123457},
				},
			},
			prepFn:  prepChunk(chunk.CSearchFiles),
			want:    2,
			wantErr: false,
		},
		{
			name: "empty file slice, is not an error",
			fields: fields{
				conn:      testDB(t),
				sessionID: 1,
			},
			args: args{
				ctx:       t.Context(),
				tx:        testDB(t),
				dbchunkID: 1,
				ff:        []slack.File{},
			},
			prepFn:  prepChunk(chunk.CSearchFiles),
			want:    0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.tx)
			}
			d := &DBP{
				conn:      tt.fields.conn,
				sessionID: tt.fields.sessionID,
				mr:        tt.fields.mr,
			}
			got, err := d.insertSearchFiles(tt.args.ctx, tt.args.tx, tt.args.dbchunkID, tt.args.in3, tt.args.ff)
			if (err != nil) != tt.wantErr {
				t.Errorf("DBP.insertSearchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DBP.insertSearchFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newUserIter(t *testing.T) {
	// fixtures
	var (
		userNotInDB            = slack.User{ID: "User Not In DB"}
		userAlreadyInDB        = slack.User{ID: "User Already In DB", Name: "original"}
		userAlreadyInDBUpdated = slack.User{ID: "User Already In DB", Name: "updated"}
	)
	// harness
	var mustUser = func(idx int, u *slack.User) repository.DBUser {
		du, err := repository.NewDBUser(1, idx, u)
		if err != nil {
			panic(err)
		}
		return *du
	}
	type args struct {
		// ur        repository.UserRepository
		tx        repository.PrepareExtContext
		dbchunkID int64
		users     []slack.User
	}
	tests := []struct {
		name   string
		args   args
		expect func(mur *mock_repository.MockUserRepository)
		want   []testutil.TestResult[*repository.DBUser]
	}{
		{
			name: "returns all users, no users in the repository",
			args: args{
				tx:        testDB(t),
				dbchunkID: 1,
				users:     fixtures.TestUsers,
			},
			expect: func(mur *mock_repository.MockUserRepository) {
				for _, u := range fixtures.TestUsers {
					mur.EXPECT().Get(gomock.Any(), gomock.Any(), u.ID).Return(repository.DBUser{}, sql.ErrNoRows)
				}
			},
			want: func() []testutil.TestResult[*repository.DBUser] {
				var ret []testutil.TestResult[*repository.DBUser]
				for i, u := range fixtures.TestUsers {
					ret = append(ret, testutil.TestResult[*repository.DBUser]{
						V:   testutil.Ptr(mustUser(i, &u)),
						Err: nil,
					})
				}
				return ret
			}(),
		},
		{
			name: "one user in the repository (no changes), one new",
			args: args{
				tx:        testDB(t),
				dbchunkID: 1,
				users: []slack.User{
					userAlreadyInDB,
					userNotInDB,
				},
			},
			expect: func(mur *mock_repository.MockUserRepository) {
				mur.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ repository.PrepareExtContext, id string) (repository.DBUser, error) {
					switch id {
					case userNotInDB.ID:
						return repository.DBUser{}, sql.ErrNoRows
					case userAlreadyInDB.ID:
						return mustUser(1, &userAlreadyInDB), nil
					}
					panic("should never get here")
				}).Times(2)
			},
			want: []testutil.TestResult[*repository.DBUser]{
				{V: testutil.Ptr(mustUser(1, &userNotInDB))},
			},
		},
		{
			name: "one user in the repository (changed), one new",
			args: args{
				tx:        testDB(t),
				dbchunkID: 1,
				users: []slack.User{
					userAlreadyInDBUpdated,
					userNotInDB,
				},
			},
			expect: func(mur *mock_repository.MockUserRepository) {
				mur.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ repository.PrepareExtContext, id string) (repository.DBUser, error) {
					switch id {
					case userNotInDB.ID:
						return repository.DBUser{}, sql.ErrNoRows
					case userAlreadyInDB.ID:
						return mustUser(1, &userAlreadyInDB), nil
					}
					panic("should never get here")
				}).Times(2)
			},
			want: []testutil.TestResult[*repository.DBUser]{
				{V: testutil.Ptr(mustUser(0, &userAlreadyInDBUpdated))},
				{V: testutil.Ptr(mustUser(1, &userNotInDB))},
			},
		},
		{
			name: "one user and already in DB",
			args: args{
				tx:        testDB(t),
				dbchunkID: 1,
				users: []slack.User{
					userAlreadyInDB,
				},
			},
			expect: func(mur *mock_repository.MockUserRepository) {
				mur.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ repository.PrepareExtContext, id string) (repository.DBUser, error) {
					switch id {
					case userAlreadyInDB.ID:
						return mustUser(1, &userAlreadyInDB), nil
					}
					panic("should never get here")
				}).Times(1)
			},
			want: []testutil.TestResult[*repository.DBUser]{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mur := mock_repository.NewMockUserRepository(ctrl)
			tt.expect(mur)
			uit := newUserIter(t.Context(), mur, tt.args.tx, tt.args.dbchunkID, tt.args.users)
			testutil.AssertIterResult(t, tt.want, uit)
		})
	}
}
