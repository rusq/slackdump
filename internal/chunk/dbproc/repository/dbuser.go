package repository

import (
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
)

type DBUser struct {
	ID          string    `db:"ID"`
	ChunkID     int64     `db:"CHUNK_ID,omitempty"`
	LoadDTTM    time.Time `db:"LOAD_DTTM,omitempty"`
	Username    string    `db:"USERNAME,omitempty"`
	DisplayName string    `db:"DISPLAY_NAME,omitempty"`
	Index       int       `db:"IDX"`
	Data        []byte    `db:"DATA"`
}

func NewDBUser(chunkID int64, n int, u *slack.User) (*DBUser, error) {
	data, err := marshal(u)
	if err != nil {
		return nil, err
	}
	return &DBUser{
		ID:          u.ID,
		ChunkID:     chunkID,
		Index:       n,
		Username:    structures.Username(u),
		DisplayName: structures.UserDisplayName(u),
		Data:        data,
	}, nil
}

func (DBUser) tablename() string {
	return "S_USER"
}

func (DBUser) columns() []string {
	return []string{"ID", "CHUNK_ID", "USERNAME", "DISPLAY_NAME", "IDX", "DATA"}
}

func (u DBUser) values() []any {
	return []any{
		u.ID,
		u.ChunkID,
		u.Username,
		u.DisplayName,
		u.Index,
		u.Data,
	}
}

type UserRepository interface {
	repository[DBUser]
}

func NewUserRepository() UserRepository {
	return newGenericRepository(DBUser{})
}
