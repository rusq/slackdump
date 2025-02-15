package repository

import (
	"context"
	"encoding/json"
	"iter"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

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
				ID:        1648085300726649,
				ChunkID:   100,
				ChannelID: "C123",
				TS:        "1648085300.726649",
				ParentID:  ptr[int64](1648085300726649),
				ThreadTS:  ptr("1648085300.726649"),
				IsParent:  true,
				Index:     222,
				NumFiles:  0,
				Text:      "This content can't be displayed.",
				Data:      minifyJSON[slack.Message](t, fixtures.BotMessageThreadParentJSON),
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
				mm: toIter([]testResult[*DBMessage]{
					{V: &DBMessage{ID: 1, ChunkID: 1, ChannelID: "C123", TS: "1.1", IsParent: false, Index: 0, NumFiles: 0, Text: "test", Data: []byte(`{"text":"test"}`)}},
					toTestResult(NewDBMessage(1, 1, "C123", fixtures.Load[*slack.Message](fixtures.SimpleMessageJSON))),
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
