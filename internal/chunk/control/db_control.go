package control

import (
	"context"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
)

type DBController struct {
	dbp *dbproc.DBP
	s   Streamer
}

func NewDB(ctx context.Context, s Streamer, proc *dbproc.DBP) (*DBController, error) {
	return &DBController{
		dbp: proc,
		s:   s,
	}, nil
}

func (c *DBController) Run(ctx context.Context, list *structures.EntityList) error {
	rec := chunk.NewCustomRecorder("dbp", c.dbp)
	sp := superprocessor{
		Conversations: rec,
		Users:         rec,
		Channels:      rec,
		WorkspaceInfo: rec,
	}
	defer rec.Close()

	return runWorkers(ctx, c.s, list, sp, Flags{})
}

func (c *DBController) Close() error {
	return c.dbp.Close()
}
