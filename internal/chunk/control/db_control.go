package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

type DBController struct {
	dbp *dbproc.DBP
	s   Streamer
	options
}

// NewDB creates a new [DBController]. Once the [Control.Close] is called it
// closes all processors, including the [dbproc.DBP].
func NewDB(ctx context.Context, s Streamer, proc *dbproc.DBP, opts ...Option) (*DBController, error) {
	d := &DBController{
		dbp: proc,
		s:   s,
		options: options{
			lg:    slog.Default(),
			tf:    &noopTransformer{},
			filer: &noopFiler{},
			avp:   &noopAvatarProc{},
		},
	}
	for _, opt := range opts {
		opt(&d.options)
	}
	return d, nil
}

func (c *DBController) newConvTransformer(ctx context.Context) *conversationTransformer {
	return &conversationTransformer{
		ctx:  ctx,
		tf:   c.tf,
		rc: c.dbp,
	}
}

func (c *DBController) Run(ctx context.Context, list *structures.EntityList) error {
	rec := chunk.NewCustomRecorder("dbp", c.dbp)
	defer rec.Close()

	sp := superprocessor{
		Conversations: processor.AppendMessenger(processor.PrependFiler(rec, c.filer), c.newConvTransformer(ctx)),
		Users:         processor.JoinUsers(c.newUserCollector(ctx), c.avp, rec),
		Channels:      rec,
		WorkspaceInfo: rec,
	}

	return runWorkers(ctx, c.s, list, sp, Flags{})
}

func (c *DBController) Close() error {
	var errs error
	if c.filer != nil {
		if err := c.filer.Close(); err != nil {
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
