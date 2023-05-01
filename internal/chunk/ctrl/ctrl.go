// Package ctrl is the Slack Stream controller.  It runs the API scraping in
// several goroutines and manages the data flow between them.
package ctrl

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/chunk/processor/dirproc"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/slack-go/slack"
)

type CtrlError struct {
	Subroutine string
	Stage      string
	Err        error
}

func (e CtrlError) Error() string {
	return fmt.Sprintf("controller error in %s on %s: %v", e.Subroutine, e.Stage, e.Err)
}

type Streamer interface {
	Conversations(ctx context.Context, proc processor.Conversations, links <-chan string, fn func(slackdump.StreamResult) error) error
	ListChannels(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error
	Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error
	WorkspaceInfo(ctx context.Context, proc processor.WorkspaceInfo) error
}

type TransformStarter interface {
	dirproc.Transformer
	StartWithUsers(ctx context.Context, users []slack.User) error
}

type Flags struct {
	MemberOnly bool
	// Type       export.ExportType
}

func Run(ctx context.Context, cd *chunk.Directory, s Streamer, tf TransformStarter, filer processor.Filer, list *structures.EntityList, flags Flags) error {
	lg := logger.FromContext(ctx)
	lg.Printf("using %s as the temporary directory", cd.Name())
	lg.Print("running export...")

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
			generator = genAPIChannel(s, cd.Name(), flags.MemberOnly)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(linkC)
			if err := generator(ctx, linkC, list); err != nil {
				errC <- CtrlError{"channel generator", "generator", err}
			}
			lg.Debug("channels done")
		}()
	}
	{
		// workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := workspaceWorker(ctx, s, cd.Name()); err != nil {
				errC <- CtrlError{"workspace", "worker", err}
			}
			lg.Debug("workspace info done")
		}()
	}
	// user goroutine
	// once all users are fetched, it triggers the transformer to start.
	{
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := userWorker(ctx, s, cd, tf); err != nil {
				errC <- CtrlError{"user", "worker", err}
				return
			}
		}()
	}
	// conversations goroutine
	{
		conv, err := dirproc.NewConversation(cd.Name(), filer, tf)
		if err != nil {
			return fmt.Errorf("error initialising conversation processor: %w", err)
		}

		pb := newProgressBar(progressbar.NewOptions(
			-1,
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSpinnerType(8)),
			lg.IsDebug(),
		)
		pb.RenderBlank()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer pb.Finish()
			defer func() {
				if err := conv.Close(); err != nil {
					errC <- CtrlError{"conversations", "close", err}
				}
			}()
			if err := conversationWorker(ctx, s, conv, pb, linkC); err != nil {
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
func genAPIChannel(s Streamer, tmpdir string, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- string, list *structures.EntityList) error {
		chIdx := list.Index()
		chanproc, err := dirproc.NewChannels(tmpdir, func(c []slack.Channel) error {
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

func userWorker(ctx context.Context, s Streamer, chunkdir *chunk.Directory, tf TransformStarter) error {
	userproc, err := dirproc.NewUsers(chunkdir.Name())
	if err != nil {
		return err
	}
	defer userproc.Close()

	if err := s.Users(ctx, userproc); err != nil {
		return fmt.Errorf("error listing users: %w", err)
	}
	if err := userproc.Close(); err != nil {
		return fmt.Errorf("error closing user processor: %w", err)
	}
	logger.FromContext(ctx).Debug("users done")
	users, err := chunkdir.Users() // load users from chunks
	if err != nil {
		return fmt.Errorf("error loading users: %w", err)
	}
	if err := tf.StartWithUsers(ctx, users); err != nil {
		return fmt.Errorf("error starting the transformer: %w", err)
	}
	return nil
}

// progresser is an interface for progress bars.
type progresser interface {
	RenderBlank() error
	Describe(description string)
	Add(num int) error
	Finish() error
}

func conversationWorker(ctx context.Context, s Streamer, proc processor.Conversations, pb progresser, links <-chan string) error {
	lg := logger.FromContext(ctx)
	if err := s.Conversations(ctx, proc, links, func(sr slackdump.StreamResult) error {
		lg.Debugf("conversations: %s", sr.String())
		pb.Describe(sr.String())
		pb.Add(1)
		return nil
	}); err != nil {
		if errors.Is(err, transform.ErrClosed) {
			return fmt.Errorf("upstream error: %w", err)
		}
		return fmt.Errorf("error streaming conversations: %w", err)
	}
	lg.Debug("conversations done")
	pb.Describe("OK")
	return nil
}

func newProgressBar(pb *progressbar.ProgressBar, debug bool) progresser {
	if debug {
		return progressbar.DefaultSilent(0)
	}
	return pb
}

func workspaceWorker(ctx context.Context, s Streamer, tmpdir string) error {
	lg := logger.FromContext(ctx)
	lg.Debug("workspaceWorker started")
	wsproc, err := dirproc.NewWorkspace(tmpdir)
	if err != nil {
		return err
	}
	defer wsproc.Close()
	if err := s.WorkspaceInfo(ctx, wsproc); err != nil {
		return err
	}
	lg.Debug("workspaceWorker done")
	return nil
}
