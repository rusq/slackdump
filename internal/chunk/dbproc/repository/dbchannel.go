package repository

import (
	"context"
	"fmt"
	"iter"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

type DBChannel struct {
	ID      string  `db:"ID"`
	ChunkID int64   `db:"CHUNK_ID"`
	Name    *string `db:"NAME"`
	Index   int     `db:"IDX"`
	Data    []byte  `db:"DATA"`
}

func NewDBChannel(chunkID int64, n int, channel *slack.Channel) (*DBChannel, error) {
	data, err := marshal(channel)
	if err != nil {
		return nil, err
	}
	return &DBChannel{
		ID:      channel.ID,
		ChunkID: chunkID,
		Name:    orNull(channel.Name != "", channel.Name),
		Index:   n,
		Data:    data,
	}, nil
}

func (c DBChannel) tablename() string {
	return "CHANNEL"
}

func (c DBChannel) columns() []string {
	return []string{"ID", "CHUNK_ID", "NAME", "IDX", "DATA"}
}

func (c DBChannel) values() []interface{} {
	return []interface{}{c.ID, c.ChunkID, c.Name, c.Index, c.Data}
}

func (c DBChannel) Val() (slack.Channel, error) {
	return unmarshalt[slack.Channel](c.Data)
}

type ChannelRepository interface {
	repository[DBChannel]
	Count(ctx context.Context, conn sqlx.QueryerContext) (int64, error)
	All(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[DBChannel, error], error)
}

type channelRepository struct {
	genericRepository[DBChannel]
}

func NewChannelRepository() ChannelRepository {
	return channelRepository{newGenericRepository(DBChannel{})}
}

func (r channelRepository) Count(ctx context.Context, conn sqlx.QueryerContext) (int64, error) {
	return r.CountType(ctx, conn, chunk.CChannelInfo)
}

func (r channelRepository) All(ctx context.Context, conn sqlx.QueryerContext) (iter.Seq2[DBChannel, error], error) {
	latest, binds := r.stmtLatest(chunk.CChannelInfo)
	stmt := "with latest as (" + latest + `) select * from latest l join channel c on c.id = l.id and c.chunk_id = l.chunk_id order by c.id`
	rows, err := conn.QueryxContext(ctx, stmt, binds...)
	if err != nil {
		return nil, fmt.Errorf("all: %w", err)
	}
	it := func(yield func(DBChannel, error) bool) {
		defer rows.Close()
		for rows.Next() {
			var c DBChannel
			if err := rows.StructScan(&c); err != nil {
				yield(DBChannel{}, fmt.Errorf("all: %w", err))
				return
			}
			if !yield(c, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(DBChannel{}, fmt.Errorf("all: %w", err))
			return
		}
	}
	return it, nil
}
