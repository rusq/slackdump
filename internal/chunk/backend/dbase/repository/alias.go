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
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type DBAlias struct {
	ChannelID string    `db:"CHANNEL_ID"`
	Alias     string    `db:"ALIAS"`
	CreatedAt time.Time `db:"CREATED_AT"`
}

//go:generate mockgen -destination=mock_repository/mock_alias.go . AliasRepository
type AliasRepository interface {
	Get(ctx context.Context, conn sqlx.QueryerContext, channelID string) (DBAlias, error)
	All(ctx context.Context, conn sqlx.QueryerContext) ([]DBAlias, error)
	Set(ctx context.Context, conn sqlx.ExtContext, channelID, alias string) error
	Delete(ctx context.Context, conn sqlx.ExtContext, channelID string) error
}

type aliasRepository struct{}

func NewAliasRepository() AliasRepository {
	return aliasRepository{}
}

func (aliasRepository) Get(ctx context.Context, conn sqlx.QueryerContext, channelID string) (DBAlias, error) {
	var a DBAlias
	stmt := rebind(conn, "SELECT CHANNEL_ID, ALIAS, CREATED_AT FROM ALIAS WHERE CHANNEL_ID = ?")
	if err := sqlx.GetContext(ctx, conn, &a, stmt, channelID); err != nil {
		return DBAlias{}, err
	}
	return a, nil
}

func (aliasRepository) All(ctx context.Context, conn sqlx.QueryerContext) ([]DBAlias, error) {
	stmt := rebind(conn, "SELECT CHANNEL_ID, ALIAS, CREATED_AT FROM ALIAS ORDER BY CHANNEL_ID")
	var aa []DBAlias
	if err := sqlx.SelectContext(ctx, conn, &aa, stmt); err != nil {
		if err == sql.ErrNoRows {
			return []DBAlias{}, nil
		}
		return nil, err
	}
	return aa, nil
}

func (aliasRepository) Set(ctx context.Context, conn sqlx.ExtContext, channelID, alias string) error {
	stmt := rebind(conn, `
INSERT INTO ALIAS (CHANNEL_ID, ALIAS)
VALUES (?, ?)
ON CONFLICT(CHANNEL_ID) DO UPDATE SET ALIAS = excluded.ALIAS
`)
	_, err := conn.ExecContext(ctx, stmt, channelID, alias)
	return err
}

func (aliasRepository) Delete(ctx context.Context, conn sqlx.ExtContext, channelID string) error {
	stmt := rebind(conn, "DELETE FROM ALIAS WHERE CHANNEL_ID = ?")
	_, err := conn.ExecContext(ctx, stmt, channelID)
	return err
}
