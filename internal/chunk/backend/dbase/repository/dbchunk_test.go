// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package repository

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func Test_chunkRepository_Insert(t *testing.T) {
	type args struct {
		ctx   context.Context
		conn  PrepareExtContext
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
				ctx:  t.Context(),
				conn: testConn(t),
				chunk: &DBChunk{
					SessionID:  1,
					UnixTS:     1234567890,
					TypeID:     chunk.CFiles,
					NumRecords: 100,
					Final:      true,
				},
			},
			prepFn: func(t *testing.T, db PrepareExtContext) {
				var r sessionRepository
				id, err := r.Insert(t.Context(), db, &Session{})
				require.NoError(t, err)
				assert.Equal(t, int64(1), id)
			},
			want:    1,
			wantErr: assert.NoError,
		},
		{
			name: "missing session",
			args: args{
				ctx:  t.Context(),
				conn: testConn(t),
				chunk: &DBChunk{
					SessionID:  1,
					UnixTS:     1234567890,
					TypeID:     chunk.CMessages,
					NumRecords: 100,
					Final:      true,
				},
			},
			want:    0,
			wantErr: assert.Error,
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

func TestDBChunk_Chunk(t *testing.T) {
	type fields struct {
		ID          int64
		SessionID   int64
		UnixTS      int64
		CreatedAt   time.Time
		TypeID      chunk.ChunkType
		NumRecords  int32
		ChannelID   *string
		SearchQuery *string
		Final       bool
		ThreadOnly  *bool
	}
	tests := []struct {
		name   string
		fields fields
		want   *chunk.Chunk
	}{
		{
			name: "messages",
			fields: fields{
				ID:         1,
				SessionID:  1,
				UnixTS:     1234567890,
				TypeID:     chunk.CMessages,
				NumRecords: 100,
				Final:      true,
				ChannelID:  ptr("C1234567890"),
			},
			want: &chunk.Chunk{
				Type:      chunk.CMessages,
				Timestamp: 1234567890,
				ChannelID: "C1234567890",
				Count:     100,
				IsLast:    true,
				Messages:  make([]slack.Message, 0, 100),
			},
		},
		{
			name: "search messages",
			fields: fields{
				ID:          1,
				SessionID:   1,
				UnixTS:      1234567890,
				TypeID:      chunk.CSearchMessages,
				NumRecords:  42,
				Final:       true,
				SearchQuery: ptr("search query"),
			},
			want: &chunk.Chunk{
				Type:           chunk.CSearchMessages,
				Timestamp:      1234567890,
				Count:          42,
				IsLast:         true,
				SearchQuery:    "search query",
				SearchMessages: make([]slack.SearchMessage, 0, 42),
			},
		},
		{
			name: "thread messages, thread only",
			fields: fields{
				ID:         1,
				SessionID:  1,
				UnixTS:     1234567890,
				TypeID:     chunk.CThreadMessages,
				NumRecords: 100,
				Final:      true,
				ChannelID:  ptr("C1234567890"),
				ThreadOnly: ptr(true),
			},
			want: &chunk.Chunk{
				Type:       chunk.CThreadMessages,
				Timestamp:  1234567890,
				ChannelID:  "C1234567890",
				Count:      100,
				IsLast:     true,
				ThreadOnly: true,
				Messages:   make([]slack.Message, 0, 100),
			},
		},
		{
			name: "thread messages, not thread only",
			fields: fields{
				ID:         1,
				SessionID:  1,
				UnixTS:     1234567890,
				TypeID:     chunk.CThreadMessages,
				NumRecords: 100,
				Final:      true,
				ChannelID:  ptr("C1234567890"),
				ThreadOnly: ptr(false),
			},
			want: &chunk.Chunk{
				Type:       chunk.CThreadMessages,
				Timestamp:  1234567890,
				ChannelID:  "C1234567890",
				Count:      100,
				IsLast:     true,
				ThreadOnly: false,
				Messages:   make([]slack.Message, 0, 100),
			},
		},
		{
			name: "channel users",
			fields: fields{
				ID:         1,
				SessionID:  1,
				UnixTS:     1234567890,
				TypeID:     chunk.CChannelUsers,
				NumRecords: 42,
				Final:      true,
			},
			want: &chunk.Chunk{
				Type:         chunk.CChannelUsers,
				Timestamp:    1234567890,
				Count:        42,
				IsLast:       true,
				ChannelUsers: make([]string, 0, 42),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DBChunk{
				ID:          tt.fields.ID,
				SessionID:   tt.fields.SessionID,
				UnixTS:      tt.fields.UnixTS,
				CreatedAt:   tt.fields.CreatedAt,
				TypeID:      tt.fields.TypeID,
				NumRecords:  tt.fields.NumRecords,
				ChannelID:   tt.fields.ChannelID,
				SearchQuery: tt.fields.SearchQuery,
				Final:       tt.fields.Final,
				ThreadOnly:  tt.fields.ThreadOnly,
			}
			if got := c.Chunk(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DBChunk.Chunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBChunk_tablename(t *testing.T) {
	type fields struct {
		ID          int64
		SessionID   int64
		UnixTS      int64
		CreatedAt   time.Time
		TypeID      chunk.ChunkType
		NumRecords  int32
		ChannelID   *string
		SearchQuery *string
		Final       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "table name",
			fields: fields{},
			want:   "CHUNK",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DBChunk{
				ID:          tt.fields.ID,
				SessionID:   tt.fields.SessionID,
				UnixTS:      tt.fields.UnixTS,
				CreatedAt:   tt.fields.CreatedAt,
				TypeID:      tt.fields.TypeID,
				NumRecords:  tt.fields.NumRecords,
				ChannelID:   tt.fields.ChannelID,
				SearchQuery: tt.fields.SearchQuery,
				Final:       tt.fields.Final,
			}
			if got := d.tablename(); got != tt.want {
				t.Errorf("DBChunk.tablename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBChunk_userkey(t *testing.T) {
	type fields struct {
		ID          int64
		SessionID   int64
		UnixTS      int64
		CreatedAt   time.Time
		TypeID      chunk.ChunkType
		NumRecords  int32
		ChannelID   *string
		SearchQuery *string
		Final       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "user key",
			want: []string{"SESSION_ID"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DBChunk{
				ID:          tt.fields.ID,
				SessionID:   tt.fields.SessionID,
				UnixTS:      tt.fields.UnixTS,
				CreatedAt:   tt.fields.CreatedAt,
				TypeID:      tt.fields.TypeID,
				NumRecords:  tt.fields.NumRecords,
				ChannelID:   tt.fields.ChannelID,
				SearchQuery: tt.fields.SearchQuery,
				Final:       tt.fields.Final,
			}
			if got := d.userkey(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DBChunk.userkey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBChunk_columns(t *testing.T) {
	tests := []struct {
		name string
		c    DBChunk
		want []string
	}{
		{
			name: "columns",
			want: []string{"SESSION_ID", "UNIX_TS", "TYPE_ID", "NUM_REC", "CHANNEL_ID", "SEARCH_QUERY", "FINAL", "THREAD_ONLY"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.columns(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DBChunk.columns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDBChunk_values(t *testing.T) {
	type fields struct {
		ID          int64
		SessionID   int64
		UnixTS      int64
		CreatedAt   time.Time
		TypeID      chunk.ChunkType
		NumRecords  int32
		ChannelID   *string
		SearchQuery *string
		Final       bool
		ThreadOnly  *bool
	}
	tests := []struct {
		name   string
		fields fields
		want   []any
	}{
		{
			name: "values",
			fields: fields{
				ID:          1,
				SessionID:   2,
				UnixTS:      3,
				CreatedAt:   time.Date(2021, 1, 2, 3, 4, 5, 6, time.UTC),
				TypeID:      chunk.CFiles,
				NumRecords:  6,
				ChannelID:   ptr("C123456789"),
				SearchQuery: new(string),
				Final:       true,
				ThreadOnly:  ptr(true),
			},
			want: []any{int64(2), int64(3), chunk.CFiles, int32(6), ptr("C123456789"), ptr(""), true, ptr(true)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := DBChunk{
				ID:          tt.fields.ID,
				SessionID:   tt.fields.SessionID,
				UnixTS:      tt.fields.UnixTS,
				CreatedAt:   tt.fields.CreatedAt,
				TypeID:      tt.fields.TypeID,
				NumRecords:  tt.fields.NumRecords,
				ChannelID:   tt.fields.ChannelID,
				SearchQuery: tt.fields.SearchQuery,
				Final:       tt.fields.Final,
				ThreadOnly:  tt.fields.ThreadOnly,
			}
			assert.Equal(t, tt.want, d.values())
		})
	}
}

func TestNewChunkRepository(t *testing.T) {
	tests := []struct {
		name string
		want ChunkRepository
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewChunkRepository(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewChunkRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChunkCount_Sum(t *testing.T) {
	tests := []struct {
		name string
		c    ChunkCount
		want int64
	}{
		{
			name: "empty",
			c:    ChunkCount{},
			want: 0,
		},
		{
			name: "values",
			c:    ChunkCount{chunk.CMessages: 1, chunk.CFiles: 2},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Sum(); got != tt.want {
				t.Errorf("ChunkCount.Sum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_chunkRepository_Count(t *testing.T) {
	type fields struct {
		genericRepository genericRepository[DBChunk]
	}
	type args struct {
		ctx         context.Context
		conn        sqlx.ExtContext
		sessionID   int64
		chunkTypeID []chunk.ChunkType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		prepFn  utilityFn
		want    ChunkCount
		wantErr bool
	}{
		{
			name: "correctly counts",
			args: args{
				ctx:       t.Context(),
				conn:      testConn(t),
				sessionID: 1,
			},
			prepFn:  prepChunk(chunk.CMessages, chunk.CThreadMessages, chunk.CFiles),
			want:    ChunkCount{chunk.CMessages: 1, chunk.CThreadMessages: 1, chunk.CFiles: 1},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepFn != nil {
				tt.prepFn(t, tt.args.conn.(PrepareExtContext))
			}
			r := chunkRepository{
				genericRepository: tt.fields.genericRepository,
			}
			got, err := r.Count(tt.args.ctx, tt.args.conn, tt.args.sessionID, tt.args.chunkTypeID...)
			if (err != nil) != tt.wantErr {
				t.Errorf("chunkRepository.Count() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chunkRepository.Count() = %v, want %v", got, tt.want)
			}
		})
	}
}
