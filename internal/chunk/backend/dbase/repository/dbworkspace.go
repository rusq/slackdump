package repository

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
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
	return DBWorkspace{}, errors.New("no workspace found")
}
