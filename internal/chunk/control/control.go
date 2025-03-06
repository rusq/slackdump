package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

type Controller struct {
	erc EncodeReferenceCloser
	s   Streamer
	options
}

// New creates a new generic [Controller], that accepts
// [EncodeReferenceCloser]. Once the [Control.Close] is called it closes all
// processors, including the [EncodeReferenceCloser].
func New(ctx context.Context, s Streamer, erc EncodeReferenceCloser, opts ...Option) (*Controller, error) {
	d := &Controller{
		erc: erc,
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

func (c *Controller) newConvTransformer(ctx context.Context) *conversationTransformer {
	return &conversationTransformer{
		ctx: ctx,
		tf:  c.tf,
		rc:  c.erc,
	}
}

func (c *Controller) Run(ctx context.Context, list *structures.EntityList) error {
	rec := chunk.NewCustomRecorder("generic", c.erc)
	defer rec.Close()

	sp := superprocessor{
		// got to do some explanation here: the order of processors is important:
		// files ==> recorder ==> transformer                             2     1            3
		Conversations: processor.AppendMessenger(processor.PrependFiler(rec, c.filer), c.newConvTransformer(ctx)),
		//                                       1                     2    3
		Users:         processor.JoinUsers(c.newUserCollector(ctx), c.avp, rec),
		Channels:      rec,
		WorkspaceInfo: rec,
	}

	return runWorkers(ctx, c.s, list, sp, c.flags)
}

func (c *Controller) Close() error {
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
	if err := c.erc.Close(); err != nil {
		errs = errors.Join(errs, fmt.Errorf("error closing database processor: %w", err))
	}
	return errs
}
