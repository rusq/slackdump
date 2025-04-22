package repository

import (
	"context"
	"encoding/json"
	"iter"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

// minifyJSON removes all spaces and newlines from the JSON string.
func minifyJSON[T any](t *testing.T, s string) []byte {
	t.Helper()
	var a T
	if err := json.Unmarshal([]byte(s), &a); err != nil {
		t.Fatalf("minifyJSON: %v", err)
	}
	b, err := marshal(a)
	if err != nil {
		t.Fatalf("minifyJSON: %v", err)
	}
	return b
}

func TestNewDBMessage(t *testing.T) {
	type args struct {
		dbchunkID int64
		idx       int
		channelID string
		msg       *slack.Message
	}
	tests := []struct {
		name    string
		args    args
		want    *DBMessage
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				dbchunkID: 100,
				idx:       222,
				channelID: "C123",
				msg:       fixtures.Load[*slack.Message](fixtures.SimpleMessageJSON),
			},
			want: &DBMessage{
				ID:        1645095505023899,
				ChunkID:   100,
				ChannelID: "C123",
				TS:        "1645095505.023899",
				IsParent:  false,
				Index:     222,
				NumFiles:  0,
				Text:      "Test message with Html chars &lt; &gt;",
				Data:      minifyJSON[slack.Message](t, fixtures.SimpleMessageJSON),
			},
			wantErr: false,
		},
		{
			name: "bot thread parent message",
			args: args{
				dbchunkID: 100,
				idx:       222,
				channelID: "C123",
				msg:       fixtures.Load[*slack.Message](fixtures.BotMessageThreadParentJSON),
			},
			want: &DBMessage{
				ID:          1648085300726649,
				ChunkID:     100,
				ChannelID:   "C123",
				TS:          "1648085300.726649",
				ParentID:    ptr[int64](1648085300726649),
				ThreadTS:    ptr("1648085300.726649"),
				LatestReply: ptr("1648085301.269949"),
				IsParent:    true,
				Index:       222,
				NumFiles:    0,
				Text:        "This content can't be displayed.",
				Data:        minifyJSON[slack.Message](t, fixtures.BotMessageThreadParentJSON),
			},
		},
		{
			name: "bot thread child message w files",
			args: args{
				dbchunkID: 100,
				idx:       222,
				channelID: "C123",
				msg:       fixtures.Load[*slack.Message](fixtures.BotMessageThreadChildJSON),
			},
			want: &DBMessage{
				ID:        1648085301269949,
				ChunkID:   100,
				ChannelID: "C123",
				TS:        "1648085301.269949",
				ParentID:  ptr[int64](1648085300726649),
				ThreadTS:  ptr("1648085300.726649"),
				IsParent:  false,
				Index:     222,
				NumFiles:  1,
				Text:      "",
				Data:      minifyJSON[slack.Message](t, fixtures.BotMessageThreadChildJSON),
			},
		},
		{
			name: "app message",
			args: args{
				dbchunkID: 100,
				idx:       222,
				channelID: "C123",
				msg:       fixtures.Load[*slack.Message](fixtures.AppMessageJSON),
			},
			want: &DBMessage{
				ID:        1586042786000100,
				ChunkID:   100,
				ChannelID: "C123",
				TS:        "1586042786.000100",
				IsParent:  false,
				Index:     222,
				NumFiles:  0,
				Text:      "",
				Data:      minifyJSON[slack.Message](t, fixtures.AppMessageJSON),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBMessage(tt.args.dbchunkID, tt.args.idx, tt.args.channelID, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newDBMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_messageRepository_Insert(t *testing.T) {
	// fixtures
	simpleDBMessage, err := NewDBMessage(1, 0, "C123", fixtures.Load[*slack.Message](fixtures.SimpleMessageJSON))
	if err != nil {
		t.Fatalf("newdbmessage: %v", err)
	}

	type args struct {
		ctx  context.Context
		conn PrepareExtContext
		m    *DBMessage
	}
	tests := []struct {
		name    string
		m       messageRepository
		args    args
		prepFn  utilityFn
		wantErr bool
		checkFn utilityFn
	}{
		{
			name: "ok",
			m:    messageRepository{},
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
				m:    simpleDBMessage,
			},
			prepFn:  prepChunk(chunk.CMessages),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn)
			}
			m := NewMessageRepository()
			if err := m.Insert(tt.args.ctx, tt.args.conn, tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.conn)
			}
		})
	}
}

func Test_messageRepository_InsertAll(t *testing.T) {
	type args struct {
		ctx   context.Context
		pconn PrepareExtContext
		mm    iter.Seq2[*DBMessage, error]
	}
	tests := []struct {
		name    string
		args    args
		prepFn  utilityFn
		want    int
		wantErr bool
		checkFn utilityFn
	}{
		{
			name: "ok",
			args: args{
				ctx:   context.Background(),
				pconn: testConn(t),
				mm: testutil.ToIter([]testutil.TestResult[*DBMessage]{
					{V: &DBMessage{ID: 1, ChunkID: 1, ChannelID: "C123", TS: "1.1", IsParent: false, Index: 0, NumFiles: 0, Text: "test", Data: []byte(`{"text":"test"}`)}},
					testutil.ToTestResult(NewDBMessage(1, 1, "C123", fixtures.Load[*slack.Message](fixtures.SimpleMessageJSON))),
				}),
			},
			prepFn:  prepChunk(chunk.CMessages),
			want:    2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.pconn)
			}
			m := NewMessageRepository()
			got, err := m.InsertAll(tt.args.ctx, tt.args.pconn, tt.args.mm)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.InsertAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("messageRepository.InsertAll() = %v, want %v", got, tt.want)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.pconn)
			}
		})
	}
}

var (
	// channel C123
	//
	// Setup:
	// Chunk Message
	// ----- -------
	//     1 +A
	//     1 +B'
	//     2 +C
	//     5 +C lead
	//     5 +-- thread msg 1
	//     5 +-- thread msg 2
	//
	msgA   = slack.Message{Msg: slack.Msg{Timestamp: "123.456", Text: "A"}}
	msgB   = slack.Message{Msg: slack.Msg{Timestamp: "124.555", Text: "B"}}
	msgB_  = slack.Message{Msg: slack.Msg{Timestamp: "124.555", Text: "B'"}}
	msgC   = slack.Message{Msg: slack.Msg{Timestamp: "125.777", Text: "C", ThreadTimestamp: "125.777"}}
	msgCt1 = slack.Message{Msg: slack.Msg{Timestamp: "125.788", Text: "C thread 1", ThreadTimestamp: "123.777"}}
	msgCt2 = slack.Message{Msg: slack.Msg{Timestamp: "125.799", Text: "C thread 2", ThreadTimestamp: "123.777"}}

	dbmA  = must(NewDBMessage(1, 0, "C123", &msgA))
	dbmB  = must(NewDBMessage(1, 1, "C123", &msgB))
	dbmB_ = must(NewDBMessage(2, 0, "C123", &msgB_))
	dbmC  = must(NewDBMessage(2, 1, "C123", &msgC))
	// chunk 5 is the CThreadMessages for the thread C
	dbmCt0 = must(NewDBMessage(5, 0, "C123", &msgC)) // message lead that we got with the thread, same as msg C.
	dbmCt1 = must(NewDBMessage(5, 1, "C123", &msgCt1))
	dbmCt2 = must(NewDBMessage(5, 2, "C123", &msgCt2))

	// channel D124
	msgX = slack.Message{Msg: slack.Msg{Timestamp: "123.456", Text: "X"}}
	msgY = slack.Message{Msg: slack.Msg{Timestamp: "124.555", Text: "Y"}}
	msgZ = slack.Message{Msg: slack.Msg{Timestamp: "125.777", Text: "Z"}}

	dbmX = must(NewDBMessage(3, 0, "D124", &msgX))
	dbmY = must(NewDBMessage(3, 1, "D124", &msgY))
	dbmZ = must(NewDBMessage(4, 0, "D124", &msgZ))
)

func messagePrepFn(t *testing.T, conn PrepareExtContext) {
	// we will use 2 chunks, one old and one new for the same channel
	// they both will have 2 messages each, such as  (A, B),(B', C)
	// where B' will be an updated version of B.
	// Also, there are messages from a different channel, X, Y, Z.
	prepChunk(chunk.CMessages, chunk.CMessages, chunk.CMessages, chunk.CMessages, chunk.CThreadMessages)(t, conn)
	mr := NewMessageRepository()
	messages := []*DBMessage{dbmA, dbmB, dbmB_, dbmC, dbmCt0, dbmCt1, dbmCt2, dbmX, dbmY, dbmZ}
	if err := mr.Insert(context.Background(), conn, messages...); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func Test_messageRepository_Count(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
	}{
		{
			name: "count the most recent messages, without duplicates",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
			},
			prepFn:  messagePrepFn,
			want:    3,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.Count(tt.args.ctx, tt.args.conn, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.Count() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("messageRepository.Count() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_messageRepository_AllForID(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    []testutil.TestResult[DBMessage]
		wantErr bool
	}{
		{
			name: "Get only channel messages for C123 (no thread, and only latest version of the message)",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
			},
			prepFn: messagePrepFn,
			want: []testutil.TestResult[DBMessage]{
				{V: *dbmA},
				{V: *dbmB_},
				{V: *dbmC},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.AllForID(tt.args.ctx, tt.args.conn, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.AllForID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func Test_messageRepository_CountThread(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		channelID string
		threadID  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
				threadID:  "123.456",
			},
			prepFn:  mkThreadSetupFn("C123", [3]bool{true, false, true}),
			want:    4,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.CountThread(tt.args.ctx, tt.args.conn, tt.args.channelID, tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.CountThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("messageRepository.CountThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	testChannelID = "C123"
	testThreadID  = "123.456"

	tmAParent  = slack.Message{Msg: slack.Msg{Timestamp: testThreadID, ThreadTimestamp: testThreadID, Text: "A"}}
	tmBChannel = slack.Message{Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: testThreadID, Text: "B", SubType: "thread_broadcast"}}
	tmB        = slack.Message{Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: testThreadID, Text: "B", SubType: "thread_broadcast"}}
	tmC        = slack.Message{Msg: slack.Msg{Timestamp: "125.000", ThreadTimestamp: testThreadID, Text: "C"}}
	tmD        = slack.Message{Msg: slack.Msg{Timestamp: "126.000", ThreadTimestamp: testThreadID, Text: "D"}}
	tmC_       = slack.Message{Msg: slack.Msg{Timestamp: "125.000", ThreadTimestamp: testThreadID, Text: "C'"}}
	// special additional message to test the reference counter
	tmXExtra = slack.Message{Msg: slack.Msg{Timestamp: "127.000", ThreadTimestamp: "127.000", Text: "X"}}
	// thread lead that has replies deleted
	tmYExtra = slack.Message{Msg: slack.Msg{Timestamp: "128.000", ThreadTimestamp: "128.000", LatestReply: structures.LatestReplyNoReplies, Text: "Y"}}

	dbtmAParent  = must(NewDBMessage(1, 0, testChannelID, &tmAParent))
	dbtmBChannel = must(NewDBMessage(1, 0, testChannelID, &tmBChannel))
	dbtmAthread  = must(NewDBMessage(2, 0, testChannelID, &tmAParent)) // A message that comes with the thread chunk.
	dbtmB        = must(NewDBMessage(2, 1, testChannelID, &tmB))
	dbtmC        = must(NewDBMessage(2, 1, testChannelID, &tmC))
	dbtmD        = must(NewDBMessage(2, 1, testChannelID, &tmD))
	dbtmC_       = must(NewDBMessage(3, 1, testChannelID, &tmC_))
	// these go into chunk 1
	dbtmXExtra = must(NewDBMessage(1, 0, testChannelID, &tmXExtra))
	dbtmYExtra = must(NewDBMessage(1, 0, testChannelID, &tmYExtra))
)

// mkThreadSetupFn creates a utility function that sets up the thread messages.
// It accepts the channelID that will be populated on the chunks, and the array
// of chunk "final" statuses.
//
// Thread setup is the following:
// chunk type_id subtype message   comment
//
//	   1       0    NULL       A   parent message
//	   1       0   bcast       B   thread broadcast in the channel - should not be included
//	   2       1    NULL       A   parent message, that is part of the thread.
//	   2       1   bcast       B   thread broadcast in the thread
//	   2       1    NULL       C   old thread message
//	   2       1    NULL       D   thread message
//	   3       1    NULL      C'   new thread message version of C.
//
//	The net result should be that we have 4 messages in the thread:
//	A, B, C', D
func mkThreadSetupFn(channelID string, chunkFinals [3]bool) utilityFn {
	return func(t *testing.T, conn PrepareExtContext) {
		prepChunkWithFinal(
			testChunk{typeID: chunk.CMessages, channelID: channelID, final: chunkFinals[0]},
			testChunk{typeID: chunk.CThreadMessages, channelID: channelID, final: chunkFinals[1]},
			testChunk{typeID: chunk.CThreadMessages, channelID: channelID, final: chunkFinals[2]},
		)(t, conn)

		mr := NewMessageRepository()
		if err := mr.Insert(context.Background(), conn,
			dbtmAParent,
			dbtmBChannel,
			dbtmB,
			dbtmC,
			dbtmD,
			dbtmC_,
		); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func Test_messageRepository_AllForThread(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		channelID string
		threadID  string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		prepareFn utilityFn
		want      []testutil.TestResult[DBMessage]
		wantErr   bool
	}{
		{
			name: "ok",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
				threadID:  "123.456",
			},
			prepareFn: mkThreadSetupFn("C123", [3]bool{true, false, true}),
			want: []testutil.TestResult[DBMessage]{
				{V: *dbtmAParent},
				{V: *dbtmB},
				{V: *dbtmC_},
				{V: *dbtmD},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepareFn != nil {
				tt.prepareFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.AllForThread(tt.args.ctx, tt.args.conn, tt.args.channelID, tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.AllForThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func TestDBMessage_Val(t *testing.T) {
	type fields struct {
		ID          int64
		ChunkID     int64
		ChannelID   string
		TS          string
		ParentID    *int64
		ThreadTS    *string
		LatestReply *string
		IsParent    bool
		Index       int
		NumFiles    int
		Text        string
		Data        []byte
	}
	tests := []struct {
		name    string
		fields  fields
		want    slack.Message
		wantErr bool
	}{
		{
			"ok",
			fields(*dbmA),
			msgA,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbm := DBMessage{
				ID:          tt.fields.ID,
				ChunkID:     tt.fields.ChunkID,
				ChannelID:   tt.fields.ChannelID,
				TS:          tt.fields.TS,
				ParentID:    tt.fields.ParentID,
				ThreadTS:    tt.fields.ThreadTS,
				IsParent:    tt.fields.IsParent,
				LatestReply: tt.fields.LatestReply,
				Index:       tt.fields.Index,
				NumFiles:    tt.fields.NumFiles,
				Text:        tt.fields.Text,
				Data:        tt.fields.Data,
			}
			got, err := dbm.Val()
			if (err != nil) != tt.wantErr {
				t.Errorf("DBMessage.Val() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DBMessage.Val() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_messageRepository_CountUnfinished(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		sessionID int64
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
	}{
		{
			name: "no unfinished threads",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: mkThreadSetupFn("C123", [3]bool{true, false, true}),
			want:   0,
		},
		{
			name: "unfinished threads",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				mkThreadSetupFn("C123", [3]bool{true, false, true})(t, conn)
				// add a new message to the thread
				mr := NewMessageRepository()
				if err := mr.Insert(context.Background(), conn, dbtmXExtra); err != nil {
					t.Fatalf("insert: %v", err)
				}
			},
			want: 1,
		},
		{
			name: "unfinished threads with deleted replies",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				mkThreadSetupFn("C123", [3]bool{true, false, true})(t, conn)
				// add a new message to the thread
				mr := NewMessageRepository()
				if err := mr.Insert(context.Background(), conn, dbtmYExtra); err != nil {
					t.Fatalf("insert: %v", err)
				}
			},
			want: 0,
		},
		// TODO: what happens if there's just a thread, and no parent?
		{
			name: "only parent messages, no threads",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				testChunks := []testChunk{
					{typeID: chunk.CMessages, channelID: "C123", final: false},
					{typeID: chunk.CMessages, channelID: "C123", final: true},
				}
				prepChunkWithFinal(testChunks...)(t, conn)
				mr := NewMessageRepository()
				// dbmC is a thread leader, but no replies are inserted.
				if err := mr.Insert(context.Background(), conn, dbmA, dbmB, dbmC); err != nil {
					t.Fatalf("insert: %v", err)
				}
			},
			want: 1, // should indicate one outstanding thread.
		},
		{
			name: "no final channel chunk, all threads are complete",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				testChunks := []testChunk{
					{typeID: chunk.CMessages, channelID: "C123", final: false},
					{typeID: chunk.CThreadMessages, channelID: "C123", final: false},
					{typeID: chunk.CThreadMessages, channelID: "C123", final: false},
				}
				prepChunkWithFinal(testChunks...)(t, conn)
				mr := NewMessageRepository()
				if err := mr.Insert(context.Background(), conn,
					dbtmAParent,
					dbtmBChannel,
					dbtmB,
					dbtmC,
					dbtmD,
					dbtmC_,
				); err != nil {
					t.Fatalf("insert: %v", err)
				}
			},
			want:    0,
			wantErr: true, // sql.ErrNoRows
		},
		{
			name: "final channel chunk appears, no unfinished threads",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
			},
			prepFn: func(t *testing.T, conn PrepareExtContext) {
				prepChunkWithFinal(
					testChunk{typeID: chunk.CMessages, channelID: "C123", final: true}, // final chunk for the channel.
					testChunk{typeID: chunk.CThreadMessages, channelID: "C123", final: false},
					testChunk{typeID: chunk.CThreadMessages, channelID: "C123", final: true},
				)(t, conn)
				mr := NewMessageRepository()
				if err := mr.Insert(context.Background(), conn,
					dbtmAParent,
					dbtmBChannel,
					dbtmB,
					dbtmC,
					dbtmD,
					dbtmC_,
				); err != nil {
					t.Fatalf("insert: %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.CountUnfinished(tt.args.ctx, tt.args.conn, tt.args.sessionID, tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.CountUnfinished() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("messageRepository.CountUnfinished() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_messageRepository_LatestMessages(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx  context.Context
		conn sqlx.QueryerContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    []testutil.TestResult[LatestMessage]
		wantErr bool
	}{
		{
			name: "returns latest messages timestamps",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
			},
			prepFn: messagePrepFn,
			want: []testutil.TestResult[LatestMessage]{
				{V: LatestMessage{ChannelID: "C123", TS: "125.777", ID: 125777}},
				{V: LatestMessage{ChannelID: "D124", TS: "125.777", ID: 125777}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.LatestMessages(tt.args.ctx, tt.args.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.LatestMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func Test_messageRepository_LatestThreads(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx  context.Context
		conn sqlx.QueryerContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    []testutil.TestResult[LatestThread]
		wantErr bool
	}{
		{
			name: "returns latest threads",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:  context.Background(),
				conn: testConn(t),
			},
			prepFn: mkThreadSetupFn("C123", [3]bool{true, false, true}),
			want: []testutil.TestResult[LatestThread]{
				{V: LatestThread{
					LatestMessage: LatestMessage{
						ChannelID: "C123",
						TS:        "126.000",
						ID:        126000,
					},
					ThreadTS: "123.456",
					ParentID: 123456,
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.LatestThreads(tt.args.ctx, tt.args.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.LatestThreads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func Test_messageRepository_Sorted(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		channelID string
		order     Order
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    []testutil.TestResult[DBMessage]
		wantErr bool
	}{
		{
			name: "returns messages in correct order",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
				order:     Desc,
			},
			prepFn: messagePrepFn,
			want: []testutil.TestResult[DBMessage]{
				{V: *dbmCt2},
				{V: *dbmCt1},
				{V: *dbmCt0},
				{V: *dbmB_},
				{V: *dbmA},
			},
			wantErr: false,
		},
		{
			name: "returns messages in ascending order",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				channelID: "C123",
				order:     Asc,
			},
			prepFn: messagePrepFn,
			want: []testutil.TestResult[DBMessage]{
				{V: *dbmA},
				{V: *dbmB_},
				{V: *dbmCt0},
				{V: *dbmCt1},
				{V: *dbmCt2},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.Sorted(tt.args.ctx, tt.args.conn, tt.args.channelID, tt.args.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.Sorted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			testutil.AssertIterResult(t, tt.want, got)
		})
	}
}

func Test_messageRepository_CountThreadOnlyParts(t *testing.T) {
	var (
		finishedThreadFn = func(t *testing.T, conn PrepareExtContext) {
			// for thread only entity list items there are no CMessage chunks, only CThreadMessages
			// and these chunks have threadOnly = true
			//
			// Sample setup:
			//
			// Thread: 123.456
			// 1. There are three chunks, 2 non-final and last one is final.
			// 2. Each chunk will have 2 messages, one is a thread message and the other is a thread lead,
			//    because API always returns the thread lead with the thread messages.
			ctx := context.Background()
			var sr sessionRepository
			sess, err := sr.Insert(ctx, conn, &Session{ID: 1, Finished: true})
			if err != nil {
				t.Fatalf("insert session: %v", err)
			}

			var bTrue = true
			// prepare and insert chunks
			chunks := [...]DBChunk{
				{ID: 1, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
				{ID: 2, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
				{ID: 3, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: true, ThreadOnly: &bTrue},
			}
			var cr chunkRepository
			for _, chunk := range chunks {
				if _, err := cr.Insert(ctx, conn, &chunk); err != nil {
					t.Fatalf("insert chunk: %v", err)
				}
			}
			var (
				parentMsg = slack.Message{Msg: slack.Msg{Timestamp: testThreadID, ThreadTimestamp: testThreadID, Text: "A"}}

				tm1 = slack.Message{Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: testThreadID, Text: "B"}}
				tm2 = slack.Message{Msg: slack.Msg{Timestamp: "125.000", ThreadTimestamp: testThreadID, Text: "C"}}
				tm3 = slack.Message{Msg: slack.Msg{Timestamp: "126.000", ThreadTimestamp: testThreadID, Text: "D"}}

				chunkMessages = [len(chunks)][2]*DBMessage{
					{
						must(NewDBMessage(1, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(1, 1, testChannelID, &tm1)),
					},
					{
						must(NewDBMessage(2, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(2, 1, testChannelID, &tm2)),
					},
					{
						must(NewDBMessage(3, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(3, 1, testChannelID, &tm3)),
					},
				}
			)

			var mr messageRepository
			for i := range chunks {
				if err := mr.Insert(ctx, conn, chunkMessages[i][:]...); err != nil {
					t.Fatalf("insert message: %v", err)
				}
			}
		}

		unfinishedThreadID = "123.789"
		unfinishedThreadFn = func(t *testing.T, conn PrepareExtContext) {
			// There are two different thread-only threads in this setup.
			// - a finished thread 123.456
			// - an unfinished thread. 123.789
			// The setup is similar to the finishedThreadFn, but the last chunk is not final for the unfinished thread.
			//
			// Goal is to test if the other thread does not influence the count.

			ctx := context.Background()
			var sr sessionRepository
			sess, err := sr.Insert(ctx, conn, &Session{ID: 1, Finished: true})
			if err != nil {
				t.Fatalf("insert session: %v", err)
			}

			var bTrue = true
			// prepare and insert chunks
			chunks := [...]DBChunk{
				// finished thread chunks
				{ID: 1, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
				{ID: 2, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
				{ID: 3, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: true, ThreadOnly: &bTrue},
				// unfinished thread chunks
				{ID: 4, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
				{ID: 5, TypeID: chunk.CThreadMessages, ChannelID: &testChannelID, SessionID: sess, Final: false, ThreadOnly: &bTrue},
			}
			var cr chunkRepository
			for _, chunk := range chunks {
				if _, err := cr.Insert(ctx, conn, &chunk); err != nil {
					t.Fatalf("insert chunk: %v", err)
				}
			}
			var (
				parentMsg = slack.Message{Msg: slack.Msg{Timestamp: testThreadID, ThreadTimestamp: testThreadID, Text: "A"}}

				// finished thread messages
				tm1 = slack.Message{Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: testThreadID, Text: "B"}}
				tm2 = slack.Message{Msg: slack.Msg{Timestamp: "125.000", ThreadTimestamp: testThreadID, Text: "C"}}
				tm3 = slack.Message{Msg: slack.Msg{Timestamp: "126.000", ThreadTimestamp: testThreadID, Text: "D"}}

				// unfinished thread messages
				parentMsg2 = slack.Message{Msg: slack.Msg{Timestamp: unfinishedThreadID, ThreadTimestamp: unfinishedThreadID, Text: "A"}}
				tm4        = slack.Message{Msg: slack.Msg{Timestamp: "124.000", ThreadTimestamp: unfinishedThreadID, Text: "B"}}
				tm5        = slack.Message{Msg: slack.Msg{Timestamp: "125.000", ThreadTimestamp: unfinishedThreadID, Text: "C"}}

				chunkMessages = [len(chunks)][2]*DBMessage{
					// finished thread
					{
						must(NewDBMessage(1, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(1, 1, testChannelID, &tm1)),
					},
					{
						must(NewDBMessage(2, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(2, 1, testChannelID, &tm2)),
					},
					{
						must(NewDBMessage(3, 0, testChannelID, &parentMsg)),
						must(NewDBMessage(3, 1, testChannelID, &tm3)),
					},
					// unfinished thread
					{
						must(NewDBMessage(4, 0, testChannelID, &parentMsg2)),
						must(NewDBMessage(4, 1, testChannelID, &tm4)),
					},
					{
						must(NewDBMessage(5, 0, testChannelID, &parentMsg2)),
						must(NewDBMessage(5, 1, testChannelID, &tm5)),
					},
				}
			)

			var mr messageRepository
			for i := range chunks {
				if err := mr.Insert(ctx, conn, chunkMessages[i][:]...); err != nil {
					t.Fatalf("insert message: %v", err)
				}
			}
		}
	)

	type fields struct {
		genericRepository genericRepository[DBMessage]
	}
	type args struct {
		ctx       context.Context
		conn      sqlx.QueryerContext
		sessionID int64
		channelID string
		threadID  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    int64
		wantErr bool
	}{
		{
			name: "finished thread",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: testChannelID,
				threadID:  testThreadID,
			},
			prepFn:  finishedThreadFn,
			want:    3,
			wantErr: false,
		},
		{
			name: "unfinished thread",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
				threadID:  unfinishedThreadID,
			},
			prepFn:  unfinishedThreadFn,
			want:    0,
			wantErr: true,
		},
		{
			name: "finished thread in unfinished thread setup",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
				threadID:  testThreadID,
			},
			prepFn:  unfinishedThreadFn,
			want:    3,
			wantErr: false,
		},
		{
			name: "no thread",
			fields: fields{
				genericRepository: genericRepository[DBMessage]{DBMessage{}},
			},
			args: args{
				ctx:       context.Background(),
				conn:      testConn(t),
				sessionID: 1,
				channelID: "C123",
				threadID:  "nonexistent",
			},
			prepFn:  finishedThreadFn,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := messageRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.CountThreadOnlyParts(tt.args.ctx, tt.args.conn, tt.args.sessionID, tt.args.channelID, tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("messageRepository.CountThreadOnlyParts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("messageRepository.CountThreadOnlyParts() = %v, want %v", got, tt.want)
			}
		})
	}
}
