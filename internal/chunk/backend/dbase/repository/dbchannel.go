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

func (c DBChannel) userkey() []string {
	return slice("ID")
}

func (c DBChannel) columns() []string {
	return []string{"ID", "CHUNK_ID", "NAME", "IDX", "DATA"}
}

func (c DBChannel) values() []any {
	return []any{c.ID, c.ChunkID, c.Name, c.Index, c.Data}
}

func (c DBChannel) Val() (slack.Channel, error) {
	return unmarshalt[slack.Channel](c.Data)
}

//go:generate mockgen -destination=mock_repository/mock_channel.go . ChannelRepository
type ChannelRepository interface {
	BulkRepository[DBChannel]
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

func (r channelRepository) Get(ctx context.Context, conn sqlx.ExtContext, id any) (DBChannel, error) {
	return r.GetType(ctx, conn, id, chunk.CChannelInfo)
}

func (r channelRepository) AllOfType(ctx context.Context, conn sqlx.QueryerContext, typeID ...chunk.ChunkType) (iter.Seq2[DBChannel, error], error) {
	return r.allOfTypeWhere(ctx, conn, queryParams{OrderBy: []string{"T.NAME"}}, typeID...)
}
