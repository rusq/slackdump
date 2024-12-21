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
	"sync"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

// Controller is the main controller of the Slack Stream.  It runs the API
// scraping in several goroutines and manages the data flow between them.
type Controller struct {
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
	// lg is the logger
	lg *slog.Logger
	// flags
	flags Flags
}

// Option is a functional option for the Controller.
type Option func(*Controller)

// WithFiler configures the controller with a file subprocessor.
func WithFiler(f processor.Filer) Option {
	return func(c *Controller) {
		c.filer = f
	}
}

// WithFlags configures the controller with flags.
func WithFlags(f Flags) Option {
	return func(c *Controller) {
		c.flags = f
	}
}

// WithTransformer configures the controller with a transformer.
func WithTransformer(tf ExportTransformer) Option {
	return func(c *Controller) {
		if tf != nil {
			c.tf = tf
		}
	}
}

// WithLogger configures the controller with a logger.
func WithLogger(lg *slog.Logger) Option {
	return func(c *Controller) {
		if lg != nil {
			c.lg = lg
		}
	}
}

// New creates a new [Controller].
func New(cd *chunk.Directory, s Streamer, opts ...Option) *Controller {
	c := &Controller{
		cd:    cd,
		s:     s,
		filer: &noopFiler{},
		tf:    &noopTransformer{},
		lg:    slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Flags are the controller flags.
type Flags struct {
	MemberOnly bool
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
	return fmt.Sprintf("controller error in subroutine %s on stage %s: %v", e.Subroutine, e.Stage, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}

func (c *Controller) Run(ctx context.Context, list *structures.EntityList) error {
	ctx, task := trace.NewTask(ctx, "Controller.Run")
	defer task.End()

	lg := c.lg.With("in", "controller.Run")

	var (
		wg    sync.WaitGroup
		errC  = make(chan error, 1)
		linkC = make(chan structures.EntityItem)
	)
	// Generator of channel IDs.
	{
		var generator linkFeederFunc
		if list.HasIncludes() {
			// inclusive export, processes only included channels.
			generator = genChFromList
		} else {
			// exclusive export (process only excludes, if any)
			generator = genChFromAPI(c.s, c.cd, c.flags.MemberOnly)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(linkC)
			defer lg.DebugContext(ctx, "channels done")

			if err := generator(ctx, linkC, list); err != nil {
				errC <- Error{"channel generator", "generator", err}
				return
			}
		}()
	}
	{
		// workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lg.DebugContext(ctx, "workspace info done")
			if err := workspaceWorker(ctx, c.s, c.cd); err != nil {
				errC <- Error{"workspace", "worker", err}
				return
			}
		}()
	}
	// user goroutine
	// once all users are fetched, it triggers the transformer to start.
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := userWorker(ctx, c.s, c.cd, c.tf); err != nil {
				errC <- Error{"user", "worker", err}
				return
			}
		}()
	}
	// conversations goroutine
	{
		conv, err := dirproc.NewConversation(c.cd, c.filer, c.tf)
		if err != nil {
			return fmt.Errorf("error initialising conversation processor: %w", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := conv.Close(); err != nil {
					errC <- Error{"conversations", "close", err}
				}
			}()
			if err := conversationWorker(ctx, c.s, conv, linkC); err != nil {
				errC <- Error{"conversations", "worker", err}
				return
			}
		}()
	}
	// sentinel
	go func() {
		wg.Wait()
		close(errC)
	}()

	// collect returned errors
	var allErr error
	for cErr := range errC {
		allErr = errors.Join(allErr, cErr)
	}
	if allErr != nil {
		return allErr
	}
	return nil
}

type linkFeederFunc func(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) error

// genChFromList feeds the channel IDs that it gets from the list to
// the links channel.  It does not fetch the channel list from the api, so
// it's blazing fast in comparison to apiChannelFeeder.  When needed, get the
// channel information from the conversations chunk files (they contain the
// chunk with channel information).
func genChFromList(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) error {
	for _, entry := range list.Index() {
		if entry.Include {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			case links <- *entry:
			}
		}
	}
	return nil
}

// genChFromAPI feeds the channel IDs that it gets from the API to the
// links channel.  It also filters out channels that are excluded in the list.
// It does not account for "included".  It ignores the thread links in the
// list.  It writes the channels to the tmpdir.
func genChFromAPI(s Streamer, cd *chunk.Directory, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) error {
		chIdx := list.Index()
		chanproc, err := dirproc.NewChannels(cd, func(c []slack.Channel) error {
		LOOP:
			for _, ch := range c {
				if memberOnly && !ch.IsMember {
					continue
				}
				for _, entry := range chIdx {
					if entry.Id == ch.ID && !entry.Include {
						continue LOOP
					}
				}
				select {
				case <-ctx.Done():
					return context.Cause(ctx)
				case links <- structures.EntityItem{
					Id:      ch.ID,
					Include: true,
				}:
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		if err := s.ListChannels(ctx, chanproc, &slack.GetConversationsParameters{Types: slackdump.AllChanTypes}); err != nil {
			return fmt.Errorf("error listing channels: %w", err)
		}
		if err := chanproc.Close(); err != nil {
			return fmt.Errorf("error closing channel processor: %w", err)
		}
		slog.DebugContext(ctx, "channels done")
		return nil
	}
}
