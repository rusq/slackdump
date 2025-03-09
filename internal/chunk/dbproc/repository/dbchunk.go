package repository

import (
	"context"
	"iter"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// DBChunk is the database representation of the Chunk.
type DBChunk struct {
	// ID is the unique identifier of the chunk within the session.
	ID          int64           `db:"ID,omitempty"`
	SessionID   int64           `db:"SESSION_ID,omitempty"`
	UnixTS      int64           `db:"UNIX_TS,omitempty"`
	CreatedAt   time.Time       `db:"CREATED_AT,omitempty"`
	TypeID      chunk.ChunkType `db:"TYPE_ID,omitempty"`
	NumRecords  int             `db:"NUM_REC"`
	ChannelID   *string         `db:"CHANNEL_ID,omitempty"`
	SearchQuery *string         `db:"SEARCH_QUERY,omitempty"`
	Final       bool            `db:"FINAL"`
}

func orZero[T any](t *T) T {
	var ret T
	if t == nil {
		return ret
	}
	return *t
}

func (c DBChunk) Chunk() *chunk.Chunk {
	cc := chunk.Chunk{
		Type:        c.TypeID,
		Timestamp:   c.UnixTS,
		ChannelID:   orZero(c.ChannelID),
		Count:       c.NumRecords,
		IsLast:      c.Final,
		SearchQuery: orZero(c.SearchQuery),
	}
	switch c.TypeID {
	case chunk.CMessages, chunk.CThreadMessages:
		cc.Messages = make([]slack.Message, 0, c.NumRecords)
	case chunk.CFiles:
		cc.Files = make([]slack.File, 0, c.NumRecords)
	case chunk.CUsers:
		cc.Users = make([]slack.User, 0, c.NumRecords)
	case chunk.CChannels, chunk.CChannelInfo:
		cc.Channels = make([]slack.Channel, 0, c.NumRecords)
	case chunk.CChannelUsers:
		cc.ChannelUsers = make([]string, 0, c.NumRecords)
	case chunk.CSearchMessages:
		cc.SearchMessages = make([]slack.SearchMessage, 0, c.NumRecords)
	case chunk.CSearchFiles:
		cc.SearchFiles = make([]slack.File, 0, c.NumRecords)
	}
	return &cc
}

func (DBChunk) tablename() string {
	return "CHUNK"
}

func (DBChunk) userkey() []string {
	// chunk is not meant to be used in "latest" queries, but in a sense, there
	// will always be a latest chunk for the session, so we can use the session
	// id as the user key. Calling latest will fail, because it relies on the
	// table having a CHUNK_ID column in the current implementation.
	return slice("SESSION_ID")
}

func (DBChunk) columns() []string {
	return []string{
		"SESSION_ID",
		"UNIX_TS",
		"TYPE_ID",
		"NUM_REC",
		"CHANNEL_ID",
		"SEARCH_QUERY",
		"FINAL",
	}
}

func (d DBChunk) values() []any {
	return []any{
		d.SessionID,
		d.UnixTS,
		d.TypeID,
		d.NumRecords,
		d.ChannelID,
		d.SearchQuery,
		d.Final,
	}
}

type ChunkRepository interface {
	// Insert should insert dbchunk into the repository and return its ID.
	Insert(ctx context.Context, conn sqlx.ExtContext, dbchunk *DBChunk) (int64, error)
	Count(ctx context.Context, conn sqlx.ExtContext, sessionID int64, chunkTypeID ...chunk.ChunkType) (ChunkCount, error)
	All(ctx context.Context, conn sqlx.ExtContext, sessionID int64, chunkTypeID ...chunk.ChunkType) (iter.Seq2[DBChunk, error], error)
}

type chunkRepository struct {
	genericRepository[DBChunk]
}

func NewChunkRepository() ChunkRepository {
	return chunkRepository{newGenericRepository(DBChunk{})}
}

func (r chunkRepository) Insert(ctx context.Context, conn sqlx.ExtContext, dbchunk *DBChunk) (int64, error) {
	stmt := r.stmtInsert()
	res, err := conn.ExecContext(ctx, stmt, dbchunk.values()...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

type ChunkCount map[chunk.ChunkType]int64

func (c ChunkCount) Sum() int64 {
	var sum int64
	for _, v := range c {
		sum += v
	}
	return sum
}

// Count returns the number of chunks in the repository for a given session and optionally of given types.
func (r chunkRepository) Count(ctx context.Context, conn sqlx.ExtContext, sessionID int64, chunkTypeID ...chunk.ChunkType) (ChunkCount, error) {
	// building
	var buf strings.Builder
	buf.WriteString("SELECT TYPE_ID, COUNT(*) FROM ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" WHERE SESSION_ID = ?")
	if len(chunkTypeID) > 0 {
		buf.WriteString(" AND TYPE_ID IN (")
		buf.WriteString(strings.Join(placeholders(chunkTypeID), ","))
		buf.WriteString(")")
	}
	buf.WriteString(" GROUP BY TYPE_ID")
	var b []any
	b = append(b, sessionID)
	for _, id := range chunkTypeID {
		b = append(b, id)
	}

	// executing
	rows, err := conn.QueryxContext(ctx, conn.Rebind(buf.String()), b...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(ChunkCount)
	var (
		typ   chunk.ChunkType
		count int64
	)
	for rows.Next() {
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, err
		}
		counts[typ] = count
	}

	return counts, rows.Err()
}

func (r chunkRepository) All(ctx context.Context, conn sqlx.ExtContext, sessionID int64, chunkTypeID ...chunk.ChunkType) (iter.Seq2[DBChunk, error], error) {
	// building
	var buf strings.Builder
	buf.WriteString("SELECT ")
	buf.WriteString(colAlias("T", append([]string{"ID"}, r.t.columns()...)...)) // ugly as
	buf.WriteString(" FROM ")
	buf.WriteString(r.t.tablename())
	buf.WriteString(" AS T WHERE SESSION_ID = ?")
	if len(chunkTypeID) > 0 {
		buf.WriteString(" AND TYPE_ID IN (")
		buf.WriteString(strings.Join(placeholders(chunkTypeID), ","))
		buf.WriteString(")")
	}
	buf.WriteString(" ORDER BY UNIX_TS")
	var b []any
	b = append(b, sessionID)
	for _, id := range chunkTypeID {
		b = append(b, id)
	}
	// executing
	rows, err := conn.QueryxContext(ctx, conn.Rebind(buf.String()), b...)
	if err != nil {
		return nil, err
	}

	iterfn := func(yield func(DBChunk, error) bool) {
		defer rows.Close()
		var c DBChunk
		for rows.Next() {
			if err := rows.StructScan(&c); err != nil {
				if !yield(DBChunk{}, err) {
					return
				}
				continue
			}
			if !yield(c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(DBChunk{}, err)
		}
	}
	return iterfn, nil
}
