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

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

type DBWorkspace struct {
	ID           int64   `db:"ID,omitempty"`
	ChunkID      int64   `db:"CHUNK_ID"`
	Team         string  `db:"TEAM"`
	User         *string `db:"USERNAME"`
	TeamID       string  `db:"TEAM_ID"`
	UserID       string  `db:"USER_ID"`
	EnterpriseID *string `db:"ENTERPRISE_ID"`
	URL          string  `db:"URL"`
	Data         []byte  `db:"DATA"`
}

func NewDBWorkspace(chunkID int64, wi *slack.AuthTestResponse) (*DBWorkspace, error) {
	data, err := marshal(wi)
	if err != nil {
		return nil, err
	}
	return &DBWorkspace{
		ChunkID:      chunkID,
		Team:         wi.Team,
		User:         orNull(wi.User != "", wi.User),
		TeamID:       wi.TeamID,
		UserID:       wi.UserID,
		EnterpriseID: orNull(wi.EnterpriseID != "", wi.EnterpriseID),
		URL:          wi.URL,
		Data:         data,
	}, nil
}

func (w DBWorkspace) tablename() string {
	return "WORKSPACE"
}

func (w DBWorkspace) userkey() []string {
	return slice("TEAM_ID")
}

func (w DBWorkspace) columns() []string {
	return []string{
		"CHUNK_ID",
		"TEAM",
		"USERNAME",
		"TEAM_ID",
		"USER_ID",
		"ENTERPRISE_ID",
		"URL",
		"DATA",
	}
}

func (w DBWorkspace) values() []any {
	return []any{
		w.ChunkID,
		w.Team,
		w.User,
		w.TeamID,
		w.UserID,
		w.EnterpriseID,
		w.URL,
		w.Data,
	}
}

func (w DBWorkspace) Val() (slack.AuthTestResponse, error) {
	return unmarshalt[slack.AuthTestResponse](w.Data)
}

//go:generate mockgen -destination=mock_repository/mock_workspace.go . WorkspaceRepository
type WorkspaceRepository interface {
	Inserter[DBWorkspace]
	Chunker[DBWorkspace]
	GetWorkspace(ctx context.Context, conn sqlx.QueryerContext) (DBWorkspace, error)
}

type workspaceRepository struct {
	genericRepository[DBWorkspace]
}

func NewWorkspaceRepository() WorkspaceRepository {
	return workspaceRepository{newGenericRepository(DBWorkspace{})}
}

// GetWorkspace returns the latest version of the workspace.
func (r workspaceRepository) GetWorkspace(ctx context.Context, conn sqlx.QueryerContext) (DBWorkspace, error) {
	it, err := r.AllOfType(ctx, conn, chunk.CWorkspaceInfo)
	if err != nil {
		return DBWorkspace{}, err
	}
	for w, err := range it {
		if err != nil {
			return DBWorkspace{}, err
		}
		// we just need one, maybe later, when there are multiple workspaces in
		// a single data base, we will need to return a slice of workspaces
		return w, nil
	}
	return DBWorkspace{}, sql.ErrNoRows
}
