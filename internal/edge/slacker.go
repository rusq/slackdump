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

package edge

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"slices"
	"sync"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/primitive"
	"github.com/rusq/slackdump/v4/internal/structures"
)

var ErrParameterMissing = errors.New("required parameter missing")

// High level functions that wrap low level calls to webclient API to return
// the data in the format close to the Slack API.

func (cl *Client) GetConversationsContext(ctx context.Context, p *slack.GetConversationsParameters) (channels []slack.Channel, _ string, err error) {
	return cl.getConversationsContext(ctx, p, false)
}

func (cl *Client) GetConversationsContextEx(ctx context.Context, p *slack.GetConversationsParameters, onlyMy bool) (channels []slack.Channel, _ string, err error) {
	return cl.getConversationsContext(ctx, p, onlyMy)
}

// group type parameter mapping
var channelTypeMap = map[string]string{
	structures.CPrivate: string(SCTPrivate),
	structures.CPublic:  string(SCTPrivateExclude),
}

type searchResult struct {
	Channels []slack.Channel
	Err      error
}

func (cl *Client) buildPipeline(resultC chan<- searchResult, chanTypes []string, onlyMy bool) []func(context.Context) {
	var (
		userbootFunc = func(ctx context.Context) {
			// getting client.userBoot information
			ub, err := cl.ClientUserBoot(ctx)
			if err != nil {
				resultC <- searchResult{Err: err}
				return
			}
			var ch = make([]slack.Channel, 0, len(ub.Channels))
			for _, c := range ub.Channels {
				ch = append(ch, c.SlackChannel())
			}
			resultC <- searchResult{Channels: ch}
		}
		imsFunc = func(ctx context.Context) {
			// collecting the IMs.
			ims, err := cl.IMList(ctx)
			if err != nil {
				resultC <- searchResult{Err: fmt.Errorf("ims: %w", err)}
				return
			}
			var ch = make([]slack.Channel, 0, len(ims))
			for _, c := range ims {
				ch = append(ch, c.SlackChannel())
			}
			resultC <- searchResult{Channels: ch}
		}
		mpimsFunc = func(ctx context.Context) {
			// collecting the MPIMs.
			mpims, err := cl.MPIMList(ctx)
			if err != nil {
				resultC <- searchResult{Err: fmt.Errorf("mpim: %w", err)}
				return
			}
			var ch = make([]slack.Channel, 0, len(mpims))
			for _, c := range mpims {
				ch = append(ch, c.SlackChannel())
			}
			resultC <- searchResult{Channels: ch}
		}
		convsFunc = func(st SearchChannelType, onlyMy bool) func(ctx context.Context) {
			return func(ctx context.Context) {
				// collecting the channels.
				ch, err := cl.SearchChannels(ctx, "", SearchChannelsParameters{
					OnlyMyChannels: onlyMy,
					ChannelTypes:   st,
				})
				if err != nil {
					resultC <- searchResult{Err: fmt.Errorf("conversations (%s) (onlymy=%t): %w", st, onlyMy, err)}
					return
				}
				resultC <- searchResult{Channels: ch}
			}
		}
	)

	stepFns := map[stepFlags]func(context.Context){
		runBoot:     userbootFunc,
		runIMs:      imsFunc,
		runMPIMs:    mpimsFunc,
		runChannels: convsFunc(SCTPrivateExclude, onlyMy),
		runPrivate:  convsFunc(SCTPrivate, onlyMy),
		runAllConvs: convsFunc(SCTAll, onlyMy),
	}

	steps := plannedSteps(chanTypes)

	pipeline := make([]func(context.Context), 0, len(steps))
	for _, step := range steps {
		if fn, ok := stepFns[step]; ok {
			pipeline = append(pipeline, fn)
		}
	}

	return pipeline
}

type stepFlags uint8

const (
	runBoot stepFlags = 1 << iota
	runIMs
	runMPIMs
	runChannels
	runPrivate
	runAllConvs
)

func (f stepFlags) String() string {
	const repr = "__*pPMIB"
	return primitive.FlagRender(uint8(f), [8]byte([]byte(repr)))
}

const maxPipelineSize = 5

func plannedSteps(chanTypes []string) []stepFlags {
	flags := pipelineFlags(chanTypes)
	slog.Debug("planned steps", "types", chanTypes, "flags", flags)

	ordered := []stepFlags{
		runBoot,
		runIMs,
		runMPIMs,
		runChannels,
		runPrivate,
		runAllConvs,
	}
	steps := make([]stepFlags, 0, len(ordered))
	for _, step := range ordered {
		if flags&step == step {
			steps = append(steps, step)
		}
	}
	return steps
}

func pipelineFlags(chanTypes []string) stepFlags {
	// treat nothing as "all"
	if len(chanTypes) == 0 {
		return runBoot | runIMs | runMPIMs | runAllConvs
	}
	// we'll be operating on a copy, as sort and compact will modify original slice
	stt := make([]string, len(chanTypes))
	copy(stt, chanTypes)

	slices.Sort(stt)
	stt = slices.Compact(stt)

	has := func(t string) bool { return slices.Contains(stt, t) }

	var flags stepFlags = 0
	if has(structures.CIM) {
		flags |= runIMs
	}
	if has(structures.CMPIM) {
		flags |= runMPIMs
	}
	if has(structures.CPrivate) && has(structures.CPublic) {
		flags |= runAllConvs + runBoot
	} else if has(structures.CPublic) {
		flags |= runChannels
	} else if has(structures.CPrivate) {
		flags |= runPrivate
	}
	return flags
}

func (cl *Client) getConversationsContext(ctx context.Context, p *slack.GetConversationsParameters, onlyMy bool) (channels []slack.Channel, _ string, err error) {
	ctx, task := trace.NewTask(ctx, "getConversationsContext")
	defer task.End()

	trace.Logf(ctx, "info", "onlyMy: %t", onlyMy)

	var resultC = make(chan searchResult, maxPipelineSize)
	pipeline := cl.buildPipeline(resultC, p.Types, onlyMy)

	var wg sync.WaitGroup
	wg.Add(len(pipeline))
	for _, f := range pipeline {
		go func(f func(context.Context)) {
			defer wg.Done()
			f(ctx)
		}(f)
	}
	go func() {
		wg.Wait()
		close(resultC)
	}()

	// create a map of channels that we have already seen
	var seenChannels = make(map[string]struct{})
	for r := range resultC {
		if r.Err != nil {
			return nil, "", r.Err
		}
		for _, c := range r.Channels {
			if _, seen := seenChannels[c.ID]; !seen {
				seenChannels[c.ID] = struct{}{}
				channels = append(channels, c)
			}
		}
	}

	// postprocessing

	// ClientCounts hopefully returns MPIM IDs that we haven't seen in the
	// user boot response.
	cr, err := cl.ClientCounts(ctx)
	if err != nil {
		return nil, "", err
	}

	// determine which mpims are already in the list, and which need to be
	// fetched
	var fetchIDs = make([]string, 0, len(cr.MPIMs))
	for _, c := range cr.MPIMs {
		if _, seen := seenChannels[c.ID]; !seen {
			fetchIDs = append(fetchIDs, c.ID)
		}
	}

	if len(fetchIDs) > 0 {
		// getting the info on any MPIMs that we haven't seen yet.
		mpims, err := cl.ConversationsGenericInfo(ctx, fetchIDs...)
		if err != nil {
			return nil, "", err
		}
		channels = append(channels, mpims...)
	}
	return channels, "", nil
}

func (cl *Client) GetUsersInConversationContext(ctx context.Context, p *slack.GetUsersInConversationParameters) (ids []string, _ string, err error) {
	if p.ChannelID == "" {
		return nil, "", ErrParameterMissing
	}
	uu, err := cl.UsersList(ctx, p.ChannelID)
	if err != nil {
		return nil, "", err
	}
	for _, u := range uu {
		ids = append(ids, u.ID)
	}
	return ids, "", nil
}

var ErrNotFound = errors.New("not found")

func (cl *Client) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	cc, err := cl.ConversationsGenericInfo(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}
	if len(cc) == 0 {
		return nil, ErrNotFound
	}
	return &cc[0], nil
}
