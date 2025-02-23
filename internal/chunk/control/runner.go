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
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

type superprocessor struct {
	processor.Conversations
	processor.Users
	processor.Channels
	processor.WorkspaceInfo
}

type linkFeederFunc func(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) error

// runWorkers coordinates the workers that fetch the data from the API and
// process it.  It runs the workers in parallel and waits for all of them to
// finish.  If any of the workers return an error, it returns the error.
func runWorkers(ctx context.Context, s Streamer, list *structures.EntityList, p superprocessor, flags Flags) error {
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
			defer lg.DebugContext(ctx, "users done")

			defer func() {
				tryClose(errC, p.Users)
			}()

			if err := userWorker(ctx, s, p.Users); err != nil {
				errC <- Error{"user", "worker", err}
				return
			}
		}()
	}
	{ // conversations goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lg.DebugContext(ctx, "conversations done")

			defer func() {
				tryClose(errC, p.Conversations)
			}()
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
		if c.memberOnly && (ch.ID[0] == 'C' && !ch.IsMember) { // skip public non-member channels
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

// genChFromAPI feeds the channel IDs that it gets from the API to the
// links channel.  It also filters out channels that are excluded in the list.
// It does not account for "included".  It ignores the thread links in the
// list.  It writes the channels to the tmpdir.
func genChFromAPI(s Streamer, chanproc processor.Channels, memberOnly bool) linkFeederFunc {
	return func(ctx context.Context, links chan<- structures.EntityItem, list *structures.EntityList) (err error) {
		genproc := newChanGenerator(links, list, memberOnly)
		proc := processor.JoinChannels(genproc, chanproc)

		defer func() {
			if err2 := proc.Close(); err != nil {
				err = errors.Join(err, err2)
			}
		}()

		if err := s.ListChannels(ctx, proc, &slack.GetConversationsParameters{Types: slackdump.AllChanTypes}); err != nil {
			return fmt.Errorf("error listing channels: %w", err)
		}
		slog.DebugContext(ctx, "channels done")
		return
	}
}
