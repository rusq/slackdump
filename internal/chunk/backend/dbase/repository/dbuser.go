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
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
)

type DBUser struct {
	ID       string `db:"ID"`
	ChunkID  int64  `db:"CHUNK_ID,omitempty"`
	Username string `db:"USERNAME,omitempty"`
	Index    int    `db:"IDX"`
	Data     []byte `db:"DATA"`
}

func NewDBUser(chunkID int64, n int, u *slack.User) (*DBUser, error) {
	data, err := marshal(u)
	if err != nil {
		return nil, err
	}
	return &DBUser{
		ID:       u.ID,
		ChunkID:  chunkID,
		Index:    n,
		Username: structures.Username(u),
		Data:     data,
	}, nil
}

func (DBUser) tablename() string {
	return "S_USER"
}

func (DBUser) userkey() []string {
	return slice("ID")
}

func (DBUser) columns() []string {
	return []string{"ID", "CHUNK_ID", "USERNAME", "IDX", "DATA"}
}

func (u DBUser) values() []any {
	return []any{
		u.ID,
		u.ChunkID,
		u.Username,
		u.Index,
		u.Data,
	}
}

func (u DBUser) Val() (slack.User, error) {
	return unmarshalt[slack.User](u.Data)
}

//go:generate mockgen -destination=mock_repository/mock_user.go . UserRepository
type UserRepository interface {
	BulkRepository[DBUser]
}

func NewUserRepository() UserRepository {
	return newGenericRepository(DBUser{})
}
