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

	"github.com/rusq/slackdump/v4/internal/chunk"
)

type DBChannelUser struct {
	ChannelID string `db:"CHANNEL_ID"`
	UserID    string `db:"USER_ID"`
	ChunkID   int64  `db:"CHUNK_ID"`
	Index     int    `db:"IDX"`
}

func NewDBChannelUser(chunkID int64, n int, channelID, userID string) (*DBChannelUser, error) {
	return &DBChannelUser{
		ChannelID: channelID,
		UserID:    userID,
		ChunkID:   chunkID,
		Index:     n,
	}, nil
}

func (DBChannelUser) tablename() string {
	return "CHANNEL_USER"
}

func (DBChannelUser) userkey() []string {
	return slice("CHANNEL_ID")
}

func (DBChannelUser) columns() []string {
	return []string{"CHANNEL_ID", "USER_ID", "CHUNK_ID", "IDX"}
}

func (c DBChannelUser) values() []any {
	return []any{c.ChannelID, c.UserID, c.ChunkID, c.Index}
}

//go:generate mockgen -destination=mock_repository/mock_chan_user.go . ChannelUserRepository
type ChannelUserRepository interface {
	BulkRepository[DBChannelUser]
	GetByChannelID(ctx context.Context, db sqlx.QueryerContext, channelID string) (iter.Seq2[DBChannelUser, error], error)
}

func NewChannelUserRepository() ChannelUserRepository {
	return channelUserRepository{newGenericRepository(DBChannelUser{})}
}

type channelUserRepository struct {
	genericRepository[DBChannelUser]
}

func (r channelUserRepository) GetByChannelID(ctx context.Context, db sqlx.QueryerContext, channelID string) (iter.Seq2[DBChannelUser, error], error) {
	qp := queryParams{
		Where:   "T.CHANNEL_ID = ?",
		Binds:   []any{channelID},
		OrderBy: slice("T.USER_ID"),
	}
	return r.allOfTypeWhere(ctx, db, qp, chunk.CChannelUsers)
}
