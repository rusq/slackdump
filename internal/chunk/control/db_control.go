package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

type DBController struct {
	dbp *dbproc.DBP
	s   Streamer
	fp  processor.Filer
	avp processor.Avatars
}

func NewDB(ctx context.Context, s Streamer, proc *dbproc.DBP, procFiles processor.Filer, procAvatar processor.Avatars) (*DBController, error) {
	return &DBController{
		dbp: proc,
		s:   s,
		fp:  procFiles,
		avp: procAvatar,
	}, nil
}

func (c *DBController) Run(ctx context.Context, list *structures.EntityList) error {
	rec := chunk.NewCustomRecorder("dbp", c.dbp)
	defer rec.Close()

	sp := superprocessor{
		Conversations: processor.PrependFiler(rec, c.fp),
		Users:         processor.JoinUsers(c.avp, rec),
		Channels:      rec,
		WorkspaceInfo: rec,
	}

	return runWorkers(ctx, c.s, list, sp, Flags{})
}

func (c *DBController) Close() error {
	var errs error
	if c.fp != nil {
		if err := c.fp.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing file processor: %w", err))
		}
	}
	if c.avp != nil {
		if err := c.avp.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing avatar processor: %w", err))
		}
	}
	if err := c.dbp.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error closing database processor: %w", err))
	}
	return errs
}
