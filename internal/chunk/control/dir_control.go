// Package control holds the implementation of the Slack Stream controller.
// It runs the API scraping in several goroutines and manages the data flow
// between them.  It records the output of the API scraper into a chunk
// directory.  It also manages the transformation of the data, if the caller
// is interested in it.
package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

// DirController is the main controller of the Slack Stream.  It runs the API
// scraping in several goroutines and manages the data flow between them.
type DirController struct {
	// chunk directory to store the scraped data.
	cd *chunk.Directory
	// streamer is the API scraper.
	s Streamer
	options
}

// NewDir creates a new [DirController]. Once the [Control.Close] is called it
// closes all file processors.
func NewDir(cd *chunk.Directory, s Streamer, opts ...Option) *DirController {
	c := &DirController{
		cd: cd,
		s:  s,
		options: options{
			lg:    slog.Default(),
			tf:    &noopTransformer{},
			filer: &noopFiler{},
			avp:   &noopAvatarProc{},
		},
	}
	for _, opt := range opts {
		opt(&c.options)
	}
	return c
}

func (c *DirController) Run(ctx context.Context, list *structures.EntityList) error {
	ctx, task := trace.NewTask(ctx, "Controller.Run")
	defer task.End()

	// prefix "d" stands for directory processor

	var dcp processor.Channels = nopChannelProcessor{}
	if !list.HasIncludes() { // all channels are included
		if p, err := dirproc.NewChannels(c.cd); err != nil {
			return Error{"channel", "init", err}
		} else {
			dcp = p
		}
	}
	dwsp, err := dirproc.NewWorkspace(c.cd)
	if err != nil {
		return Error{"workspace", "init", err}
	}
	dconv, err := dirproc.NewConversation(c.cd, c.filer, c.tf, dirproc.WithRecordFiles(c.flags.RecordFiles))
	if err != nil {
		return fmt.Errorf("error initialising conversation processor: %w", err)
	}
	dusr, err := dirproc.NewUsers(c.cd)
	if err != nil {
		return Error{"user", "init", err}
	}
	userproc := processor.JoinUsers(
		c.newUserCollector(ctx),
		dusr,
		c.avp,
	)

	mp := superprocessor{
		Channels:      dcp,
		WorkspaceInfo: dwsp,
		Users:         userproc,
		Conversations: dconv,
	}

	return runWorkers(ctx, c.s, list, mp, c.flags)
}

// Close closes the controller and all its file processors.
func (c *DirController) Close() error {
	var errs error
	if c.avp != nil {
		if err := c.avp.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing avatar processor: %w", err))
		}
	}
	if c.filer != nil {
		if err := c.filer.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing file processor: %w", err))
		}
	}
	return errs
}
