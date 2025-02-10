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
	"io"
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
	// avp is avatar downloader (subprocessor), if not configured with options,
	// it's a noop, as it's not necessary
	avp processor.Avatars
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

// WithAvatarProcessor configures the controller with an avatar downloader.
func WithAvatarProcessor(avp processor.Avatars) Option {
	return func(c *Controller) {
		c.avp = avp
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

// New creates a new [Controller]. Once the [Control.Close] is called it
// closes all file processors.
func New(cd *chunk.Directory, s Streamer, opts ...Option) *Controller {
	c := &Controller{
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

type multiprocessor struct {
	processor.Conversations
	processor.Users
	processor.Channels
	processor.WorkspaceInfo
}

func tryClose(errC chan<- error, a any) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("recovered from panic", "recover", r)
		}
	}()
	if cl, ok := a.(io.Closer); ok {
		if err := cl.Close(); err != nil {
			select {
			case errC <- fmt.Errorf("error closing %T: %w", a, err):
			default:
				// give up
			}
		}
	}
}

type nopChannelProcessor struct{}

func (nopChannelProcessor) Channels(ctx context.Context, ch []slack.Channel) error {
	return nil
}

func (c *Controller) Run(ctx context.Context, list *structures.EntityList) error {
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
	userproc := joinUserProcessors(collector, dup, c.avp)

	mp := multiprocessor{
		Channels:      chanproc,
		WorkspaceInfo: wsproc,
		Users:         userproc,
		Conversations: conv,
	}

	return runWorkers(ctx, c.s, list, mp, c.flags)
}

func runWorkers(ctx context.Context, s Streamer, list *structures.EntityList, p multiprocessor, flags Flags) error {
	ctx, task := trace.NewTask(ctx, "runWorkers")
	defer task.End()

	lg := slog.With("in", "runWorkers")

	var (
		wg    sync.WaitGroup
		errC  = make(chan error, 1)
		linkC = make(chan structures.EntityItem)
	)
	{ // generator of channel IDs
		var generator linkFeederFunc
		if list.HasIncludes() {
			// inclusive export, processes only included channels.
			generator = genChFromList
		} else {
			// exclusive export (process only excludes, if any)
			generator = genChFromAPI(s, p.Channels, flags.MemberOnly)
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
	{ // workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lg.DebugContext(ctx, "workspace info done")

			defer func() {
				tryClose(errC, p.WorkspaceInfo)
			}()
			if err := workspaceWorker(ctx, s, p.WorkspaceInfo); err != nil {
				errC <- Error{"workspace", "worker", err}
				return
			}
		}()
	}
	{ // user goroutine
		// once all users are fetched, it triggers the transformer to start.
		wg.Add(1)
		go func() {
			defer wg.Done()

			defer func() {
				tryClose(errC, p.Users)
			}()

			if err := userWorker2(ctx, s, p.Users); err != nil {
				errC <- Error{"user", "worker", err}
				return
			}
		}()
	}
	{ // conversations goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			tryClose(errC, p.Conversations)
			if err := conversationWorker(ctx, s, p.Conversations, linkC); err != nil {
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

// func (c *Controller) Run2(ctx context.Context, list *structures.EntityList) error {
// 	ctx, task := trace.NewTask(ctx, "Controller.Run")
// 	defer task.End()
//
// 	lg := c.lg.With("in", "controller.Run")
//
// 	var (
// 		wg    sync.WaitGroup
// 		errC  = make(chan error, 1)
// 		linkC = make(chan structures.EntityItem)
// 	)
// 	{ // generator of channel IDs
// 		var generator linkFeederFunc
// 		if list.HasIncludes() {
// 			// inclusive export, processes only included channels.
// 			generator = genChFromList
// 		} else {
// 			// exclusive export (process only excludes, if any)
// 			generator = genChFromAPI(c.s, c.cd, c.flags.MemberOnly)
// 		}
//
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			defer close(linkC)
// 			defer lg.DebugContext(ctx, "channels done")
//
// 			if err := generator(ctx, linkC, list); err != nil {
// 				errC <- Error{"channel generator", "generator", err}
// 				return
// 			}
// 		}()
// 	}
// 	{ // workspace info
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			defer lg.DebugContext(ctx, "workspace info done")
//
// 			wsproc, err := dirproc.NewWorkspace(c.cd)
// 			if err != nil {
// 				errC <- Error{"workspace", "init", err}
// 				return
// 			}
// 			defer wsproc.Close()
//
// 			if err := workspaceWorker(ctx, c.s, wsproc); err != nil {
// 				errC <- Error{"workspace", "worker", err}
// 				return
// 			}
// 		}()
// 	}
// 	{ // user goroutine
// 		// once all users are fetched, it triggers the transformer to start.
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
//
// 			collector := &userCollector{
// 				users: make([]slack.User, 0, 100),
// 				ts:    c.tf,
// 				ctx:   ctx,
// 			}
// 			dup, err := dirproc.NewUsers(c.cd)
// 			if err != nil {
// 				errC <- Error{"user", "init", err}
// 				return
// 			}
// 			userproc := joinUserProcessors(collector, dup, c.avp)
// 			if err := userWorker2(ctx, c.s, userproc); err != nil {
// 				userproc.Close()
// 				errC <- Error{"user", "worker", err}
// 				return
// 			}
// 			if err := userproc.Close(); err != nil {
// 				errC <- Error{"user", "close", err}
// 				return
// 			}
// 		}()
// 	}
// 	{ // conversations goroutine
// 		conv, err := dirproc.NewConversation(c.cd, c.filer, c.tf, dirproc.WithRecordFiles(c.flags.RecordFiles))
// 		if err != nil {
// 			return fmt.Errorf("error initialising conversation processor: %w", err)
// 		}
//
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			defer func() {
// 				if err := conv.Close(); err != nil {
// 					errC <- Error{"conversations", "close", err}
// 				}
// 			}()
// 			if err := conversationWorker(ctx, c.s, conv, linkC); err != nil {
// 				errC <- Error{"conversations", "worker", err}
// 				return
// 			}
// 		}()
// 	}
// 	// sentinel
// 	go func() {
// 		wg.Wait()
// 		close(errC)
// 	}()
//
// 	// collect returned errors
// 	var allErr error
// 	for cErr := range errC {
// 		allErr = errors.Join(allErr, cErr)
// 	}
// 	if allErr != nil {
// 		return allErr
// 	}
// 	return nil
// }

// Close closes the controller and all its file processors.
func (c *Controller) Close() error {
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

type chanGenerator struct {
	links      chan<- structures.EntityItem
	list       *structures.EntityList
	memberOnly bool
	idx        map[string]*structures.EntityItem
}

func newChanGenerator(links chan<- structures.EntityItem, list *structures.EntityList, memberOnly bool) *chanGenerator {
	return &chanGenerator{
		links:      links,
		list:       list,
		memberOnly: memberOnly,
		idx:        list.Index(),
	}
}

func (c *chanGenerator) Channels(ctx context.Context, ch []slack.Channel) error {
LOOP:
	for _, ch := range ch {
		if c.memberOnly && !ch.IsMember {
			continue
		}
		for _, entry := range c.idx {
			if entry.Id == ch.ID && !entry.Include {
				continue LOOP
			}
		}
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case c.links <- structures.EntityItem{Id: ch.ID, Include: true}:
		}
	}
	return nil
}

type channelProcessor struct {
	processors []processor.Channels
}

func joinChannelProcessors(procs ...processor.Channels) *channelProcessor {
	return &channelProcessor{processors: procs}
}

func (c *channelProcessor) Channels(ctx context.Context, ch []slack.Channel) error {
	for _, p := range c.processors {
		if err := p.Channels(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func (c *channelProcessor) Close() error {
	var errs error
	for i := len(c.processors) - 1; i >= 0; i-- {
		if closer, ok := c.processors[i].(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// genChFromAPI feeds the channel IDs that it gets from the API to the
// links channel.  It also filters out channels that are excluded in the list.
// It does not account for "included".  It ignores the thread links in the
// list.  It writes the channels to the tmpdir.
func genChFromAPI(s Streamer, chanproc processor.Channels, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) (err error) {
		genproc := newChanGenerator(links, list, memberOnly)
		proc := joinChannelProcessors(genproc, chanproc)

		defer func() {
			err = proc.Close()
		}()

		if err := s.ListChannels(ctx, proc, &slack.GetConversationsParameters{Types: slackdump.AllChanTypes}); err != nil {
			return fmt.Errorf("error listing channels: %w", err)
		}
		slog.DebugContext(ctx, "channels done")
		return
	}
}
