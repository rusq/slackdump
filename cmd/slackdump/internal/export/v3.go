package export

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/chunk/processor/expproc"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/subproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

type ExportError struct {
	Subroutine string
	Stage      string
	Err        error
}

func (e ExportError) Error() string {
	return fmt.Sprintf("export error in %s on %s: %v", e.Subroutine, e.Stage, e.Err)
}

func exportV3(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, list *structures.EntityList, options export.Config) error {
	lg := logger.FromContext(ctx)

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}
	chunkdir, err := chunk.OpenDir(tmpdir)
	if err != nil {
		return err
	}
	if !lg.IsDebug() {
		defer chunkdir.RemoveAll()
	}
	tf, err := transform.NewExport(ctx, fsa, tmpdir, transform.WithBufferSize(1000), transform.WithMsgUpdateFunc(subproc.ExportTokenUpdateFn(options.ExportToken)))
	if err != nil {
		return fmt.Errorf("failed to create transformer: %w", err)
	}
	defer tf.Close()

	lg.Printf("using %s as the temporary directory", tmpdir)
	lg.Print("running export...")

	var (
		wg    sync.WaitGroup
		s     = sess.Stream()
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
			generator = genAPIChannel(s, tmpdir, options.MemberOnly)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(linkC)
			if err := generator(ctx, linkC, list); err != nil {
				errC <- ExportError{"channel generator", "generator", err}
			}
			lg.Debug("channels done")
		}()
	}
	{
		// workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := workspaceWorker(ctx, s, tmpdir); err != nil {
				errC <- ExportError{"workspace", "worker", err}
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
			if err := userWorker(ctx, s, tmpdir, chunkdir, tf); err != nil {
				errC <- ExportError{"user", "worker", err}
				return
			}
		}()
	}
	// conversations goroutine
	{
		// starting the downloader
		var sdl subproc.Downloader
		if options.Type == export.TNoDownload || !cfg.DumpFiles {
			sdl = subproc.NoopDownloader{}
		} else {
			dl := downloader.New(sess.Client(), fsa, downloader.WithLogger(lg))
			dl.Start(ctx)
			defer dl.Stop()
			sdl = dl
		}

		conv, err := expproc.NewConversation(tmpdir, subproc.NewExport(options.Type, sdl), tf)
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
					errC <- ExportError{"conversations", "close", err}
				}
			}()
			if err := conversationWorker(ctx, s, conv, pb, linkC); err != nil {
				errC <- ExportError{"conversations", "worker", err}
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
		allErr = errors.Join(err, cErr)
	}
	if allErr != nil {
		return allErr
	}

	// at this point no goroutines are running, we are safe to assume that
	// everything we need is in the chunk directory.
	if err := tf.WriteIndex(); err != nil {
		return err
	}
	lg.Debug("index written")
	lg.Printf("conversations export finished, chunk files in: %s", tmpdir)
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
func genAPIChannel(s *slackdump.Stream, tmpdir string, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- string, list *structures.EntityList) error {
		chIdx := list.Index()
		chanproc, err := expproc.NewChannels(tmpdir, func(c []slack.Channel) error {
			for _, ch := range c {
				if memberOnly && !ch.IsMember {
					continue
				}
				if include, ok := chIdx[ch.ID]; ok && !include {
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

func userWorker(ctx context.Context, s *slackdump.Stream, tmpdir string, chunkdir *chunk.Directory, tf *transform.Export) error {
	userproc, err := expproc.NewUsers(tmpdir)
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

func conversationWorker(ctx context.Context, s *slackdump.Stream, proc processor.Conversations, pb progresser, links <-chan string) error {
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

func workspaceWorker(ctx context.Context, s *slackdump.Stream, tmpdir string) error {
	lg := logger.FromContext(ctx)
	lg.Debug("workspaceWorker started")
	wsproc, err := expproc.NewWorkspace(tmpdir)
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
