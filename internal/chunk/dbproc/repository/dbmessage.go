package repository

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
)

type DBMessage struct {
	ID        int64   `db:"ID,omitempty"`
	ChunkID   int64   `db:"CHUNK_ID,omitempty"`
	ChannelID string  `db:"CHANNEL_ID"`
	TS        string  `db:"TS"`
	ParentID  *int64  `db:"PARENT_ID,omitempty"`
	ThreadTS  *string `db:"THREAD_TS,omitempty"`
	IsParent  bool    `db:"IS_PARENT"`
	Index     int     `db:"IDX"`
	NumFiles  int     `db:"NUM_FILES"`
	Text      string  `db:"TXT"`
	Data      []byte  `db:"DATA"`
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
		ID:        ts,
		ChunkID:   dbchunkID,
		ChannelID: channelID,
		TS:        msg.Timestamp,
		ParentID:  parentID,
		ThreadTS:  orNull(msg.ThreadTimestamp != "", msg.ThreadTimestamp),
		IsParent:  structures.IsThreadStart(msg),
		Index:     idx,
		NumFiles:  len(msg.Files),
		Text:      msg.Text,
		Data:      data,
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
	}
}

func (dbm DBMessage) Val() (slack.Message, error) {
	return unmarshalt[slack.Message](dbm.Data)
}

type MessageRepository interface {
	Inserter[DBMessage]
	// Count returns the number of messages in a channel.
	Count(ctx context.Context, conn sqlx.QueryerContext, channelID string) (int64, error)
	// AllForID returns all messages in a channel.
	AllForID(ctx context.Context, conn sqlx.QueryerContext, channelID string) (iter.Seq2[DBMessage, error], error)
	// CountThread returns the number of messages in a thread.
	CountThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (int64, error)
	// AllForThread returns all messages in a thread.
	AllForThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (iter.Seq2[DBMessage, error], error)
}

type messageRepository struct {
	genericRepository[DBMessage]
}

func NewMessageRepository() MessageRepository {
	return messageRepository{newGenericRepository(DBMessage{})}
}

func (r messageRepository) Count(ctx context.Context, conn sqlx.QueryerContext, channelID string) (int64, error) {
	return r.countTypeWhere(ctx, conn, chunk.CMessages, queryParams{Where: "CHANNEL_ID = ?", Binds: []any{channelID}})
}

func (r messageRepository) AllForID(ctx context.Context, conn sqlx.QueryerContext, channelID string) (iter.Seq2[DBMessage, error], error) {
	return r.allOfTypeWhere(ctx, conn, chunk.CMessages, queryParams{Where: "CHANNEL_ID = ?", Binds: []any{channelID}, UserKeyOrder: true})
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
	return r.countTypeWhere(ctx, conn, CTypeAny, queryParams{Where: r.threadCond(), Binds: []any{channelID, parentID}})
}

func (r messageRepository) AllForThread(ctx context.Context, conn sqlx.QueryerContext, channelID, threadID string) (iter.Seq2[DBMessage, error], error) {
	parentID, err := fasttime.TS2int(threadID)
	if err != nil {
		return nil, fmt.Errorf("allForThread fasttime: %w", err)
	}
	return r.allOfTypeWhere(ctx, conn, CTypeAny, queryParams{Where: r.threadCond(), Binds: []any{channelID, parentID}, UserKeyOrder: true})
}
