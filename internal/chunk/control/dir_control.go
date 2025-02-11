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

	"github.com/rusq/slack"

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
	// tf is the transformer of the chunk data. It may not be necessary, if
	// caller is not interested in transforming the data.
	tf ExportTransformer
	// files subprocessor, if not configured with options, it's a noop, as
	// it's not necessary for all use cases.
	filer processor.Filer
	// avp is avatar downloader (subprocessor), if not configured with options,
	// it's a noop, as it's not necessary
	avp processor.Avatars
	// lg is the logger
	lg *slog.Logger
	// flags
	flags Flags
}

// Option is a functional option for the Controller.
type Option func(*DirController)

// WithFiler configures the controller with a file subprocessor.
func WithFiler(f processor.Filer) Option {
	return func(c *DirController) {
		c.filer = f
	}
}

// WithAvatarProcessor configures the controller with an avatar downloader.
func WithAvatarProcessor(avp processor.Avatars) Option {
	return func(c *DirController) {
		c.avp = avp
	}
}

// WithFlags configures the controller with flags.
func WithFlags(f Flags) Option {
	return func(c *DirController) {
		c.flags = f
	}
}

// WithTransformer configures the controller with a transformer.
func WithTransformer(tf ExportTransformer) Option {
	return func(c *DirController) {
		if tf != nil {
			c.tf = tf
		}
	}
}

// WithLogger configures the controller with a logger.
func WithLogger(lg *slog.Logger) Option {
	return func(c *DirController) {
		if lg != nil {
			c.lg = lg
		}
	}
}

// NewDir creates a new [DirController]. Once the [Control.Close] is called it
// closes all file processors.
func NewDir(cd *chunk.Directory, s Streamer, opts ...Option) *DirController {
	c := &DirController{
		cd: cd,
		s:  s,
		lg: slog.Default(),

		tf: &noopTransformer{},

		filer: &noopFiler{},
		avp:   &noopAvatarProc{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Flags are the controller flags.
type Flags struct {
	MemberOnly  bool
	RecordFiles bool
}

// Error is a controller error.
type Error struct {
	// Subroutine is the name of the subroutine that failed.
	Subroutine string
	// Stage is the stage of the subroutine that failed.
	Stage string
	// Err is the error that caused the failure.
	Err error
}

func (e Error) Error() string {
	return fmt.Sprintf("error in subroutine %s on stage %s: %v", e.Subroutine, e.Stage, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}

func (c *DirController) Run(ctx context.Context, list *structures.EntityList) error {
	ctx, task := trace.NewTask(ctx, "Controller.Run")
	defer task.End()

	var chanproc processor.Channels = nopChannelProcessor{}
	if !list.HasIncludes() {
		var err error
		chanproc, err = dirproc.NewChannels(c.cd)
		if err != nil {
			return Error{"channel", "init", err}
		}
	}
	wsproc, err := dirproc.NewWorkspace(c.cd)
	if err != nil {
		return Error{"workspace", "init", err}
	}
	conv, err := dirproc.NewConversation(c.cd, c.filer, c.tf, dirproc.WithRecordFiles(c.flags.RecordFiles))
	if err != nil {
		return fmt.Errorf("error initialising conversation processor: %w", err)
	}
	collector := &userCollector{
		users: make([]slack.User, 0, 100),
		ts:    c.tf,
		ctx:   ctx,
	}
	dup, err := dirproc.NewUsers(c.cd)
	if err != nil {
		collector.Close()
		return Error{"user", "init", err}
	}
	userproc := processor.JoinUsers(collector, dup, c.avp)

	mp := superprocessor{
		Channels:      chanproc,
		WorkspaceInfo: wsproc,
		Users:         userproc,
		Conversations: conv,
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

