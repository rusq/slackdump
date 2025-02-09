package repository

import (
	"encoding/json"
	"time"

	"github.com/rusq/slack"
)

type DBUser struct {
	ID       string    `db:"ID"`
	ChunkID  int64     `db:"CHUNK_ID,omitempty"`
	LoadDTTM time.Time `db:"LOAD_DTTM,omitempty"`
	Index    int       `db:"IDX"`
	Data     []byte    `db:"DATA"`
}

func NewDBUser(chunkID int64, n int, u *slack.User) (*DBUser, error) {
	data, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	return &DBUser{
		ID:      u.ID,
		ChunkID: chunkID,
		Index:   n,
		Data:    data,
	}, nil
}

func (*DBUser) Table() string {
	return "S_USER"
}

func (*DBUser) Columns() []string {
	return []string{"ID", "CHUNK_ID", "IDX", "DATA"}
}
func (u *DBUser) Values() []any {
	return []any{
		u.ID,
		u.ChunkID,
		u.Index,
		u.Data,
	}
}

type UserRepository interface {
	repository[*DBUser]
}

func NewUserRepository() UserRepository {
	return newGenericRepository[*DBUser]()
}
