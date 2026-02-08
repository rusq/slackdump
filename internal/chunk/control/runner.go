// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
	"golang.org/x/sync/errgroup"

	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
)

// Flags are the controller flags.
type Flags struct {
	// MemberOnly is the flag to fetch only those channels where the user is a
	// member.
	MemberOnly bool
	// RecordFiles instructs directory processor to record the files as chunks.
	RecordFiles bool
	// Refresh is to fetch additional channels from the API in addition to
	// those provided in the list.  It's useful when the list is
	// incomplete or outdated.
	Refresh bool
	// ChannelUsers is the flag to fetch only users involved in the channel,
	// and skip fetching of all users.
	ChannelUsers bool
	// ChannelTypes is the list of channel types to fetch.  If empty, all
	// channel types are fetched.
	ChannelTypes []string
	// IncludeLabels requests API to include the labels for the custom fields.
	// works only with ChannelUsers. Server may throttle requests hard.
	IncludeLabels bool
}

// Error is a controller error.
type Error struct {
	// Subroutine is the name of the subroutine that failed.
	Subroutine string
	// Stage is the stage of the subroutine that failed.
	Stage Stage
	// Err is the error that caused the failure.
	Err error
}

// Stage is the stage controller that failed.
type Stage string

const (
	// StgGenerator is the generator stage.
	StgGenerator Stage = "generator"
	// StgWorker is the worker stage.
	StgWorker Stage = "worker"
)

func (e Error) Error() string {
	return fmt.Sprintf("error in subroutine %s on stage %s: %v", e.Subroutine, e.Stage, e.Err)
}

func (e Error) Unwrap() error {
	return e.Err
}

// superprocessor is a combination of all processors necessary for scraping messages.
type superprocessor struct {
	processor.Conversations
	processor.Users
	processor.Channels
	processor.WorkspaceInfo
}

func newGenerator(s Streamer, p superprocessor, flags Flags, list *structures.EntityList) generator {
	// choose your fighter
	if flags.Refresh {
		// refresh the given list with the channels from the API.
		return &combinedGenerator{
			s:       s,
			p:       p.Channels,
			chTypes: flags.ChannelTypes,
		}
	} else if list.HasIncludes() {
		// inclusive export, processes only included channels.
		return &listGen{}
	} else {
		// exclusive export (process only excludes, if any)
		return &apiGenerator{
			s:          s,
			p:          p.Channels,
			memberOnly: flags.MemberOnly,
			chTypes:    flags.ChannelTypes,
		}
	}
}

// runWorkers coordinates the workers that fetch the data from the API and
// process it.  It runs the workers in parallel and waits for all of them to
// finish.  If any of the workers return an error, it returns the error.
func runWorkers(ctx context.Context, s Streamer, list *structures.EntityList, p superprocessor, flags Flags) error {
	ctx, task := trace.NewTask(ctx, "runWorkers")
	defer task.End()

	lg := slog.With("in", "runWorkers")

	var (
		wg   sync.WaitGroup
		errC = make(chan error, 1)
	)

	{ // workspace info
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer lg.DebugContext(ctx, "workspace info done")

			defer func() {
				tryClose(errC, p.WorkspaceInfo)
			}()
			if err := workspaceWorker(ctx, s, p.WorkspaceInfo); err != nil {
				errC <- Error{"workspace", StgWorker, err}
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
				errC <- Error{"user", StgWorker, err}
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
			gen := newGenerator(s, p, flags, list)
			listC, wait := gen.Generate(ctx, errC, list)
			defer wait() // sync with the generator
			if err := conversationWorker(ctx, s, p.Conversations, listC); err != nil {
				errC <- Error{"conversations", StgWorker, err}
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

// generator is something that generates the channel IDs.
type generator interface {
	// Generate should take the context, error channel, and the list of channels
	// and return a channel of channel IDs.  It should close the channel when
	// done.
	Generate(ctx context.Context, errC chan<- error, list *structures.EntityList) (listC <-chan structures.EntityItem, wait func())
}

// apiGenerator feeds the channel IDs that it gets from the API to the links
// channel.  It also filters out channels that are excluded in the list.  It
// does not account for "included".  It ignores the thread links in the list.
type apiGenerator struct {
	s          Streamer
	p          processor.Channels
	memberOnly bool
	chTypes    []string
}

func (g *apiGenerator) Generate(ctx context.Context, errC chan<- error, list *structures.EntityList) (<-chan structures.EntityItem, func()) {
	if len(g.chTypes) == 0 {
		g.chTypes = slackdump.AllChanTypes
	}
	lg := slog.With("types", g.chTypes)
	lg.DebugContext(ctx, "API channel generator starting")
	linksC := make(chan structures.EntityItem)
	emitErr := errEmitter(errC, "api channel generator", StgGenerator)
	done := make(chan struct{})

	go func() {
		defer close(linksC)
		defer close(done)

		genproc := newChanFilter(linksC, list, g.memberOnly)
		joined := processor.JoinChannels(genproc, g.p)
		defer func() {
			if err := joined.Close(); err != nil {
				emitErr(fmt.Errorf("error closing processor: %w", err))
			}
		}()

		//
		// ListChannels -> joined.Channels -(-> (filters) -)-> output to entity item channel
		//
		if err := g.s.ListChannels(ctx, joined, &slack.GetConversationsParameters{Types: g.chTypes}); err != nil {
			emitErr(fmt.Errorf("error listing channels: %w", err))
			return
		}
		slog.DebugContext(ctx, "channels done")
	}()
	return linksC, func() { <-done }
}

// combinedGenerator combines the list and channels from the API.  It first sends
// the channels from the list, then fetches the rest from the API.  It does not
// account for "included".  It ignores the thread links in the list.
type combinedGenerator struct {
	s       Streamer
	p       processor.Channels
	chTypes []string
}

func (g *combinedGenerator) Generate(ctx context.Context, errC chan<- error, list *structures.EntityList) (<-chan structures.EntityItem, func()) {
	if len(g.chTypes) == 0 {
		g.chTypes = slackdump.AllChanTypes
	}
	links := make(chan structures.EntityItem)
	emitErr := errEmitter(errC, "combined channel generator", StgGenerator)
	done := make(chan struct{})

	go func() {
		defer close(links)
		defer close(done)

		// TODO: this can be made more efficient, if the processed is pre-cooked.
		//       API fetching can happen separately and fan in the entries. Drawback
		//       is that it will be harder to maintain the order of the channels.

		proc := &combinedChannels{
			output:    links,
			processed: make(map[string]struct{}),
		}
		// joined processor will take care of duplicates and will send only
		// the channels that are not in the processed list.
		joined := processor.JoinChannels(proc, g.p)
		defer func() {
			if err := joined.Close(); err != nil {
				emitErr(fmt.Errorf("error closing processor: %w", err))
			}
		}()

		// process the list first
		for entry := range list.C(ctx) {
			select {
			case <-ctx.Done():
				emitErr(context.Cause(ctx))
				return
			case links <- entry:
				// mark as processed
				proc.processed[entry.Id] = struct{}{}
			}
		}

		// process the rest, if any
		if err := g.s.ListChannels(ctx, joined, &slack.GetConversationsParameters{Types: g.chTypes}); err != nil {
			emitErr(fmt.Errorf("error listing channels: %w", err))
			return
		}
	}()
	return links, func() { <-done }
}

// listGen is a simplest generator that just emits the channels from the list
// passed to it.
type listGen struct{}

func (g *listGen) Generate(ctx context.Context, _ chan<- error, list *structures.EntityList) (<-chan structures.EntityItem, func()) {
	return list.C(ctx), func() {}
}

// supersearcher is a combination of all processors necessary for searching.
type supersearcher struct {
	processor.WorkspaceInfo
	processor.MessageSearcher
	processor.FileSearcher
}

func runSearch(ctx context.Context, s Streamer, sp supersearcher, stype SearchType, query string) error {
	if stype == srchUnknown || stype > (SMessages|SFiles) {
		return errors.New("internal: unknown search type")
	}
	lg := slog.With("query", query)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return workspaceWorker(ctx, s, sp.WorkspaceInfo)
	})

	// conditionally start search workers

	if stype&SMessages != 0 {
		eg.Go(func() error {
			lg.InfoContext(ctx, "searching messages")
			return searchMsgWorker(ctx, s, sp.MessageSearcher, query)
		})
	}
	if stype&SFiles != 0 {
		eg.Go(func() error {
			lg.InfoContext(ctx, "searching files")
			return searchFileWorker(ctx, s, sp.FileSearcher, query)
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error searching: %w", err)
	}
	return nil
}
