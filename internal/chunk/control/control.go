// Package ctrl is the Slack Stream controller.  It runs the API scraping in
// several goroutines and manages the data flow between them.
package control

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/processor"
	"github.com/slack-go/slack"
)

// Controller is the main controller of the Slack Stream.  It runs the API
// scraping in several goroutines and manages the data flow between them.
type Controller struct {
	// chunk directory to store the data.
	cd *chunk.Directory
	// streamer is the API scraper.
	s Streamer
	// transformer, it may not be necessary, if caller is not interested in
	// transforming the data.
	tf TransformStarter
	// files subprocessor, if not configured with options, it's a noop, as
	// it's not necessary for all use cases.
	pfiles processor.Filer
	// lg is the logger
	lg logger.Interface
	// resultFn is a list of functions to be called on each result that
	// comes from the streamer.
	resultFn []func(slackdump.StreamResult) error

	// flags
	flags Flags
}

// Option is a functional option for the Controller.
type Option func(*Controller)

// WithFiler configures the controller with a file subprocessor.
func WithFiler(f processor.Filer) Option {
	return func(c *Controller) {
		c.pfiles = f
	}
}

// WithFlags configures the controller with flags.
func WithFlags(f Flags) Option {
	return func(c *Controller) {
		c.flags = f
	}
}

// WithResultFn configures the controller with a result function.
func WithResultFn(fn func(slackdump.StreamResult) error) Option {
	return func(c *Controller) {
		c.resultFn = append(c.resultFn, fn)
	}
}

// WithTransformer configures the controller with a transformer.
func WithTransformer(tf TransformStarter) Option {
	return func(c *Controller) {
		if tf != nil {
			c.tf = tf
		}
	}
}

// WithLogger configures the controller with a logger.
func WithLogger(lg logger.Interface) Option {
	return func(c *Controller) {
		if lg != nil {
			c.lg = lg
		}
	}
}

// New creates a new [Controller].
func New(cd *chunk.Directory, s Streamer, opts ...Option) *Controller {
	c := &Controller{
		cd:     cd,
		s:      s,
		pfiles: &noopFiler{},
		tf:     &noopTransformer{},
		lg:     logger.Default,
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

// CtrlError is a controller error.
type CtrlError struct {
	// Subroutine is the name of the subroutine that failed.
	Subroutine string
	// Stage is the stage of the subroutine that failed.
	Stage string
	// Err is the error that caused the failure.
	Err error
}

func (e CtrlError) Error() string {
	return fmt.Sprintf("controller error in %s on %s: %v", e.Subroutine, e.Stage, e.Err)
}

func (c *Controller) Run(ctx context.Context, list *structures.EntityList) error {
	ctx, task := trace.NewTask(logger.NewContext(ctx, c.lg), "Controller.Run")
	defer task.End()

	lg := c.lg

	var (
		wg    sync.WaitGroup
		errC  = make(chan error, 1)
		linkC = make(chan string)
	)
	// Generator of channel IDs.
	{
		var generator linkFeederFunc
		if list.HasIncludes() {
			// inclusive export, processes only included channels.
			generator = genListChannel
		} else {
			// exclusive export (process only excludes, if any)
			generator = genAPIChannel(c.s, c.cd, c.flags.MemberOnly)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(linkC)
			defer lg.Debug("channels done")

			if err := generator(ctx, linkC, list); err != nil {
				errC <- CtrlError{"channel generator", "generator", err}
				return
			}
		}()
	}
	{
		// workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lg.Debug("workspace info done")
			if err := workspaceWorker(ctx, c.s, c.cd); err != nil {
				errC <- CtrlError{"workspace", "worker", err}
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
				errC <- CtrlError{"user", "worker", err}
				return
			}
		}()
	}
	// conversations goroutine
	{
		conv, err := dirproc.NewConversation(c.cd, c.pfiles, c.tf)
		if err != nil {
			return fmt.Errorf("error initialising conversation processor: %w", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := conv.Close(); err != nil {
					errC <- CtrlError{"conversations", "close", err}
				}
			}()
			if err := conversationWorker(ctx, c.s, conv, linkC, c.resultFn...); err != nil {
				errC <- CtrlError{"conversations", "worker", err}
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

type linkFeederFunc func(ctx context.Context, links chan<- string, list *structures.EntityList) error

// genListChannel feeds the channel IDs that it gets from the list to
// the links channel.  It does not fetch the channel list from the api, so
// it's blazing fast in comparison to apiChannelFeeder.  When needed, get the
// channel information from the conversations chunk files (they contain the
// chunk with channel information).
func genListChannel(ctx context.Context, links chan<- string, list *structures.EntityList) error {
	for _, ch := range list.Include {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case links <- ch:
		}
	}
	return nil
}

// genAPIChannel feeds the channel IDs that it gets from the API to the
// links channel.  It also filters out channels that are excluded in the list.
// It does not account for "included".  It ignores the thread links in the
// list.  It writes the channels to the tmpdir.
func genAPIChannel(s Streamer, cd *chunk.Directory, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- string, list *structures.EntityList) error {
		chIdx := list.Index()
		chanproc, err := dirproc.NewChannels(cd, func(c []slack.Channel) error {
			for _, ch := range c {
				if memberOnly && !ch.IsMember {
					continue
				}
				if chIdx.IsExcluded(ch.ID) {
					continue
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case links <- ch.ID:

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
		logger.FromContext(ctx).Debug("channels done")
		return nil
	}
}
