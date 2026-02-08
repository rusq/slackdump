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
package slackdump

// In this file: messages related code.

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/types"
)

// Dump dumps messages or threads specified by link. link can be one of the
// following:
//
//   - Channel URL        - i.e. https://ora600.slack.com/archives/CHM82GF99
//   - Thread URL         - i.e. https://ora600.slack.com/archives/CHM82GF99/p1577694990000400
//   - ChannelID          - i.e. CHM82GF99
//   - ChannelID:ThreadTS - i.e. CHM82GF99:1577694990.000400
//
// oldest and latest timestamps set a timeframe  within which the messages
// should be retrieved, also one can provide process functions.
func (s *Session) Dump(ctx context.Context, link string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return nil, err
	}
	if s.cfg.dumpFiles {
		fn, cancelFn, err := s.newFileProcessFn(ctx, sl.Channel, s.limiter(network.NoTier))
		if err != nil {
			return nil, err
		}
		defer cancelFn()
		processFn = append(processFn, fn)
	}

	return s.dump(ctx, sl, oldest, latest, processFn...)
}

// DumpAll dumps all messages.  See description of Dump for what can be provided
// in link.
func (s *Session) DumpAll(ctx context.Context, link string) (*types.Conversation, error) {
	return s.Dump(ctx, link, time.Time{}, time.Time{})
}

// DumpRaw dumps all messages, but does not account for any options
// defined, such as DumpFiles, instead, the caller must hassle about any
// processFns they want to apply.
func (s *Session) DumpRaw(ctx context.Context, link string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return nil, err
	}
	return s.dump(ctx, sl, oldest, latest, processFn...)
}

func (s *Session) dump(ctx context.Context, sl structures.SlackLink, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dump")
	defer task.End()
	trace.Logf(ctx, "info", "sl: %q", sl)
	if !sl.IsValid() {
		return nil, errors.New("invalid link")
	}

	if sl.IsThread() {
		return s.dumpThreadAsConversation(ctx, sl, oldest, latest, processFn...)
	} else {
		return s.dumpChannel(ctx, sl.Channel, oldest, latest, processFn...)
	}
}

// dumpChannel fetches messages from the conversation identified by channelID.
// processFn will be called on each batch of messages returned from API.
func (s *Session) dumpChannel(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dumpMessages")
	defer task.End()

	if channelID == "" {
		return nil, errors.New("channelID is empty")
	}

	trace.Logf(ctx, "info", "channelID: %q, oldest: %s, latest: %s", channelID, oldest, latest)

	var (
		// slack rate limits are per method, so we're safe to use different limiters for different methods.
		convLimiter   = s.limiter(network.Tier3)
		threadLimiter = s.limiter(network.Tier3)
	)

	// add thread dumper.  It should go first, because it populates message
	// chunk with thread messages.
	pfns := append([]ProcessFunc{s.newThreadProcessFn(ctx, threadLimiter, oldest, latest)}, processFn...)

	var (
		messages   []types.Message
		cursor     string
		fetchStart = time.Now()
	)
	for i := 1; ; i++ {
		var resp *slack.GetConversationHistoryResponse
		reqStart := time.Now()
		if err := network.WithRetry(ctx, convLimiter, s.cfg.limits.Tier3.Retries, func(ctx context.Context) error {
			var err error
			trace.WithRegion(ctx, "GetConversationHistoryContext", func() {
				resp, err = s.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
					ChannelID: channelID,
					Cursor:    cursor,
					Limit:     s.cfg.limits.Request.Conversations,
					Oldest:    structures.FormatSlackTS(oldest),
					Latest:    structures.FormatSlackTS(latest),
					Inclusive: true,
				})
			})
			if err != nil {
				return fmt.Errorf("failed to dump channel %s: %w", channelID, err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		if !resp.Ok {
			trace.Logf(ctx, "error", "not ok, api error=%s", resp.Error)
			return nil, fmt.Errorf("response not ok, slack error: %s", resp.Error)
		}

		chunk := types.ConvertMsgs(resp.Messages)

		results, err := runProcessFuncs(chunk, channelID, pfns...)
		if err != nil {
			return nil, err
		}

		messages = append(messages, chunk...)

		s.log.InfoContext(ctx, "messages", "request", i, "fetched", len(resp.Messages), "total", len(messages),
			"process results", results,
			"speed", float64(len(resp.Messages))/time.Since(reqStart).Seconds(),
			"avg", float64(len(messages))/time.Since(fetchStart).Seconds(),
		)

		if !resp.HasMore {
			s.log.InfoContext(ctx, "messages fetch complete", "total", len(messages))
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	types.SortMessages(messages)

	name, err := s.getChannelName(ctx, s.limiter(network.Tier3), channelID)
	if err != nil {
		return nil, err
	}

	return &types.Conversation{Name: name, Messages: messages, ID: channelID}, nil
}

func (s *Session) getChannelName(ctx context.Context, l *rate.Limiter, channelID string) (string, error) {
	// get channel name
	var ci *slack.Channel
	if err := network.WithRetry(ctx, l, s.cfg.limits.Tier3.Retries, func(ctx context.Context) error {
		var err error
		ci, err = s.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{ChannelID: channelID})
		return err
	}); err != nil {
		return "", err
	}
	return ci.Name, nil
}
