package repository

import (
	"context"
	"iter"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
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
