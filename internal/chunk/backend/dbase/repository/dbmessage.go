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
	"iter"
	"runtime/trace"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
)

type DBMessage struct {
	ID          int64   `db:"ID,omitempty"`
	ChunkID     int64   `db:"CHUNK_ID,omitempty"`
	ChannelID   string  `db:"CHANNEL_ID"`
	TS          string  `db:"TS"`
	ParentID    *int64  `db:"PARENT_ID,omitempty"`
	ThreadTS    *string `db:"THREAD_TS,omitempty"`
	LatestReply *string `db:"LATEST_REPLY,omitempty"`
	IsParent    bool    `db:"IS_PARENT"`
	Index       int     `db:"IDX"`
	NumFiles    int     `db:"NUM_FILES"`
	Text        string  `db:"TXT"`
	Data        []byte  `db:"DATA"`
}

func NewDBMessage(dbchunkID int64, idx int, channelID string, msg *slack.Message) (*DBMessage, error) {
	ts, err := fasttime.TS2int(msg.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("insertMessages fasttime: %w", err)
	}
	data, err := marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("insertMessages marshal: %w", err)
	}
	var parentID *int64
	if msg.ThreadTimestamp != "" {
		if parID, err := fasttime.TS2int(msg.ThreadTimestamp); err != nil {
			return nil, fmt.Errorf("insertMessages fasttime thread: %w", err)
		} else {
			parentID = &parID
		}
	}

	dbm := DBMessage{
		ID:          ts,
		ChunkID:     dbchunkID,
		ChannelID:   channelID,
		TS:          msg.Timestamp,
		ParentID:    parentID,
		ThreadTS:    orNull(msg.ThreadTimestamp != "", msg.ThreadTimestamp),
		LatestReply: orNull(msg.LatestReply != "", msg.LatestReply),
		IsParent:    structures.IsThreadStart(msg),
		Index:       idx,
		NumFiles:    len(msg.Files),
		Text:        msg.Text,
		Data:        data,
	}
	return &dbm, nil
}

func (dbm DBMessage) tablename() string {
	return "MESSAGE"
}

func (dbm DBMessage) userkey() []string {
	return slice("ID")
}

func (dbm DBMessage) columns() []string {
	return []string{
		"ID",
		"CHUNK_ID",
		"CHANNEL_ID",
		"TS",
		"PARENT_ID",
		"THREAD_TS",
		"IS_PARENT",
		"IDX",
		"NUM_FILES",
		"TXT",
		"DATA",
		"LATEST_REPLY",
	}
}

func (dbm DBMessage) values() []any {
	return []any{
		dbm.ID,
		dbm.ChunkID,
		dbm.ChannelID,
		dbm.TS,
		dbm.ParentID,
		dbm.ThreadTS,
		dbm.IsParent,
		dbm.Index,
		dbm.NumFiles,
		dbm.Text,
		dbm.Data,
		dbm.LatestReply,
	}
}

func (dbm DBMessage) Val() (slack.Message, error) {
	return unmarshalt[slack.Message](dbm.Data)
}

// MessageRepository provides an interface for working with messages in the
// database.
//
//go:generate mockgen -destination=mock_repository/mock_message.go . MessageRepository
type MessageRepository interface {
	Inserter[DBMessage]
	Chunker[DBMessage]
	Getter[DBMessage]
	// Count returns the number of messages in a channel.
	Count(ctx context.Context, conn sqlx.QueryerContext, channelID string) (int64, error)
	// AllForID returns all messages in a channel.
	AllForID(ctx context.Context, conn sqlx.QueryerContext, channelID string) (iter.Seq2[DBMessage, error], error)
	// CountThread returns the number of messages in a thread.
	CountThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (int64, error)
	// AllForThread returns all messages in a thread, including parent message.
	AllForThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (iter.Seq2[DBMessage, error], error)
	// Sorted returns all thread and channel messages in ascending or descending
	// time order.
	Sorted(ctx context.Context, conn sqlx.QueryerContext, channelID string, order Order) (iter.Seq2[DBMessage, error], error)
	// CountUnfinished returns the number of unfinished threads in a channel.
	CountUnfinished(ctx context.Context, conn sqlx.QueryerContext, sessionID int64, channelID string) (int64, error)
	// CountThreadOnlyParts should return the number of parts in a complete
	// thread-only thread. If an unfinished or non-existent thread is
	// requested, it should return the sql.ErrNoRows error.
	CountThreadOnlyParts(ctx context.Context, conn sqlx.QueryerContext, sessionID int64, channelID, threadID string) (int64, error)
	// LatestMessages returns the latest message in each channel.
	LatestMessages(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[LatestMessage, error], error)
	// LatestThreads returns the latest thread message in each channel.
	LatestThreads(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[LatestThread, error], error)
}

var _ MessageRepository = messageRepository{}

type messageRepository struct {
	genericRepository[DBMessage]
}

func NewMessageRepository() MessageRepository {
	return messageRepository{newGenericRepository(DBMessage{})}
}

const threadOnlyCondition = " AND ((CH.TYPE_ID=0 AND (CH.THREAD_ONLY=FALSE OR CH.THREAD_ONLY IS NULL)) OR (CH.TYPE_ID=1 AND CH.THREAD_ONLY=TRUE AND T.IS_PARENT=TRUE))"

func (r messageRepository) Count(ctx context.Context, conn sqlx.QueryerContext, channelID string) (int64, error) {
	return r.countTypeWhere(
		ctx,
		conn,
		queryParams{
			Where: "T.CHANNEL_ID = ?" + threadOnlyCondition,
			Binds: []any{channelID}},
		chunk.CMessages, chunk.CThreadMessages,
	)
}

func (r messageRepository) AllForID(ctx context.Context, conn sqlx.QueryerContext, channelID string) (iter.Seq2[DBMessage, error], error) {
	return r.allOfTypeWhere(
		ctx,
		conn,
		queryParams{
			Where:        "T.CHANNEL_ID = ?" + threadOnlyCondition,
			Binds:        []any{channelID},
			UserKeyOrder: true,
		},
		chunk.CMessages, chunk.CThreadMessages,
	)
}

// threadCond returns a condition for selecting messages that are part of a
// thread with additional filtering of thread_broadcast subtype.
func (r messageRepository) threadCond() string {
	var buf strings.Builder
	buf.WriteString("T.CHANNEL_ID = ? AND T.PARENT_ID = ? ")
	buf.WriteString("AND ( JSON_EXTRACT(T.DATA, '$.subtype') IS NULL ")
	buf.WriteString("OR (JSON_EXTRACT(T.DATA, '$.subtype') = 'thread_broadcast' AND CH.TYPE_ID = 1 )")
	buf.WriteString("   ) ")
	return buf.String()
}

func (r messageRepository) CountThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (int64, error) {
	parentID, err := fasttime.TS2int(threadID)
	if err != nil {
		return 0, fmt.Errorf("countThread fasttime: %w", err)
	}
	return r.countTypeWhere(ctx, conn, queryParams{Where: r.threadCond(), Binds: []any{channelID, parentID}}, chunk.CMessages, chunk.CThreadMessages)
}

func (r messageRepository) AllForThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (iter.Seq2[DBMessage, error], error) {
	parentID, err := fasttime.TS2int(threadID)
	if err != nil {
		return nil, fmt.Errorf("allForThread fasttime: %w", err)
	}
	return r.allOfTypeWhere(ctx, conn, queryParams{Where: r.threadCond(), Binds: []any{channelID, parentID}, UserKeyOrder: true}, chunk.CMessages, chunk.CThreadMessages)
}

func (r messageRepository) Sorted(ctx context.Context, conn sqlx.QueryerContext, channelID string, order Order) (iter.Seq2[DBMessage, error], error) {
	return r.allOfTypeWhere(ctx, conn, queryParams{Where: "T.CHANNEL_ID = ?", Binds: []any{channelID}, OrderBy: []string{"T.ID" + order.String()}}, chunk.CMessages, chunk.CThreadMessages)
}

func (r messageRepository) CountUnfinished(ctx context.Context, conn sqlx.QueryerContext, sessionID int64, channelID string) (int64, error) {
	ctx, task := trace.NewTask(ctx, "CountUnfinished")
	defer task.End()
	const stmt = "SELECT REF_COUNT FROM V_UNFINISHED_CHANNELS WHERE SESSION_ID = ? AND CHANNEL_ID = ?"
	var count int64
	if err := conn.QueryRowxContext(ctx, rebind(conn, stmt), sessionID, channelID).Scan(&count); err != nil {
		return 0, fmt.Errorf("countUnfinished query: %w", err)
	}
	return count, nil
}

func (r messageRepository) CountThreadOnlyParts(ctx context.Context, conn sqlx.QueryerContext, sessionID int64, channelID, threadID string) (int64, error) {
	ctx, task := trace.NewTask(ctx, "CountUnfinishedThreads")
	defer task.End()
	const stmt = "SELECT PARTS FROM V_THREAD_ONLY_THREADS WHERE SESSION_ID = ? AND CHANNEL_ID = ? AND THREAD_TS = ?"
	var count int64
	if err := conn.QueryRowxContext(ctx, rebind(conn, stmt), sessionID, channelID, threadID).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountThreadOnlyParts query: %w", err)
	}
	return count, nil
}

type LatestMessage struct {
	ChannelID string `db:"CHANNEL_ID"`
	TS        string `db:"TS"`
	ID        int64  `db:"ID"`
}

type LatestThread struct {
	LatestMessage
	ThreadTS string `db:"THREAD_TS"`
	ParentID int64  `db:"PARENT_ID"`
}

func (r messageRepository) LatestMessages(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[LatestMessage, error], error) {
	const stmt = "SELECT CHANNEL_ID, TS, ID FROM V_LATEST_MESSAGE"
	return query[LatestMessage](ctx, conn, stmt)
}

func (r messageRepository) LatestThreads(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[LatestThread, error], error) {
	const stmt = "SELECT CHANNEL_ID, TS, ID, THREAD_TS, PARENT_ID FROM V_LATEST_THREAD"
	return query[LatestThread](ctx, conn, stmt)
}

func query[T any](ctx context.Context, conn sqlx.QueryerContext, stmt string, binds ...any) (iter.Seq2[T, error], error) {
	rows, err := conn.QueryxContext(ctx, stmt, binds...)
	if err != nil {
		return nil, err
	}
	iterFn := func(yield func(T, error) bool) {
		defer rows.Close()
		var t T
		for rows.Next() {
			if err := rows.StructScan(&t); err != nil {
				yield(t, err)
				return
			}
			if !yield(t, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(t, err)
			return
		}
	}
	return iterFn, nil
}
