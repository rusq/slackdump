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
package stream

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"sync"
	"time"

	"github.com/rusq/slack"
	"golang.org/x/sync/errgroup"

	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
)

// SyncConversations fetches the conversations from the link which can be a
// channelID, channel URL, thread URL or a link in Slackdump format.
func (cs *Stream) SyncConversations(ctx context.Context, proc processor.Conversations, items ...structures.EntityItem) error {
	lg := slog.With("links", items)
	return cs.ConversationsCB(ctx, proc, items, func(sr Result) error {
		lg.DebugContext(ctx, "stream: finished processing", "result", sr)
		return nil
	})
}

func (cs *Stream) ConversationsCB(ctx context.Context, proc processor.Conversations, items []structures.EntityItem, cb func(Result) error) error {
	ctx, task := trace.NewTask(ctx, "channelStream.Conversations")
	defer task.End()

	lg := slog.With("links", items)
	cs.resultFn = append(cs.resultFn, cb)

	itemC := make(chan structures.EntityItem, 1)
	go func() {
		defer close(itemC)
		for _, l := range items {
			select {
			case itemC <- l:
			case <-ctx.Done():
				return
			}
		}
		lg.DebugContext(ctx, "stream: sent link count", "len", len(items))
	}()

	if err := cs.Conversations(ctx, proc, itemC); err != nil {
		return err
	}
	return nil
}

// Conversations fetches the conversations from the links channel.  The link
// sent on that channel can be a channelID, channel URL, thread URL or a link
// in Slackdump format.  fn is called for each result (channel messages, or
// thread messages).  The fact that fn was called for channel messages, does
// not mean that all threads for that channel were already processed.  Each
// last thread result is marked with StreamResult.IsLast.  The caller must
// track the number of threads processed for each channel, and when the thread
// result with IsLast is received, the caller can assume that all threads and
// messages for that channel have been processed.  For example, see
// [cmd/slackdump/internal/export/expproc].
func (cs *Stream) Conversations(ctx context.Context, proc processor.Conversations, items <-chan structures.EntityItem) error {
	ctx, task := trace.NewTask(ctx, "AsyncConversations")
	defer task.End()

	// create channels
	chansC := make(chan request, msgChanSz)
	threadsC := make(chan request, threadChanSz)

	resultsC := make(chan Result, resultSz)

	var wg sync.WaitGroup
	{
		// channel worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs.channelWorker(ctx, proc, resultsC, threadsC, chansC)
			// we close threads here, instead of the main loop, because we want to
			// close it after all the threads are sent by channels.
			close(threadsC)
			trace.Log(ctx, "async", "channel worker done")
		}()
	}
	{
		// thread worker
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs.threadWorker(ctx, proc, resultsC, threadsC)
			trace.Log(ctx, "async", "thread worker done")
		}()
	}
	{
		// main loop
		wg.Add(1)
		go func() {
			defer trace.Log(ctx, "async", "main loop done")
			defer wg.Done()
			defer close(chansC)
			for {
				select {
				case <-ctx.Done():
					resultsC <- Result{Type: RTMain, Err: context.Cause(ctx)}
					return
				case item, more := <-items:
					if !more {
						return
					}
					if err := processLink(chansC, threadsC, item); err != nil {
						resultsC <- Result{Type: RTMain, Err: fmt.Errorf("item error: %q: %w", item.String(), err)}
					}
				}
			}
		}()
	}
	go func() {
		// sentinel waits for all the workers to finish, then closes the error
		// channel.
		wg.Wait()
		close(resultsC)
		trace.Log(ctx, "async", "sentinel done")
	}()

	// result processing.
	for res := range resultsC {
		if err := res.Err; err != nil {
			trace.Logf(ctx, "error", "type: %s, chan_id: %s, thread_ts: %s, error: %s", res.Type, res.ChannelID, res.ThreadTS, err.Error())
			if errors.Is(err, errChanNotFound) && !cs.failChnlNotFnd {
				slog.WarnContext(ctx, "channel not found, skipping", "channel_id", res.ChannelID)
				continue
			}
			trace.Logf(ctx, "error", "type: %s, chan_id: %s, thread_ts: %s, error: %s", res.Type, res.ChannelID, res.ThreadTS, err.Error())
			return &res // res implements Error
		}
		for _, fn := range cs.resultFn {
			if err := fn(res); err != nil {
				return fmt.Errorf("result %s, callback error: %w", res, err)
			}
		}
	}
	trace.Log(ctx, "info", "complete")
	return nil
}

// processLink parses the link and sends it to the appropriate output channel.
func processLink(channels chan<- request, threads chan<- request, link structures.EntityItem) error {
	sl, err := structures.ParseLink(link.Id)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return fmt.Errorf("invalid slack link: %s", link.Id)
	}
	if sl.IsThread() {
		threads <- request{sl: &sl, threadOnly: true, Oldest: link.Oldest, Latest: link.Latest}
	} else {
		channels <- request{sl: &sl, Oldest: link.Oldest, Latest: link.Latest}
	}
	return nil
}

type request struct {
	sl *structures.SlackLink
	// threadOnly indicates that this is the thread directly requested by the
	// user, and not a thread that was found in the channel.
	threadOnly bool
	Oldest     time.Time
	Latest     time.Time
}

// channel fetches the channel data as defined in req, calling callback function for each API response.
func (cs *Stream) channel(ctx context.Context, req request, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "channel")
	defer task.End()

	var cursor string
	for {
		var resp *slack.GetConversationHistoryResponse
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
			var apiErr error
			r := trace.StartRegion(ctx, "GetConversationHistoryContext")
			defer r.End()
			resp, apiErr = cs.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
				ChannelID: req.sl.Channel,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Conversations,
				Oldest:    structures.FormatSlackTS(structures.NVLTime(req.Oldest, cs.oldest)),
				Latest:    structures.FormatSlackTS(structures.NVLTime(req.Latest, cs.latest)),
				Inclusive: cs.inclusive,
			})
			return apiErr
		}); err != nil {
			return err
		}
		if !resp.Ok {
			trace.Logf(ctx, "error", "not ok, api error=%s", resp.Error)
			return fmt.Errorf("response not ok, slack error: %s", resp.Error)
		}

		r := trace.StartRegion(ctx, "channel_callback")
		err := callback(resp.Messages, !resp.HasMore)
		r.End()
		if err != nil {
			// lg.Printf("channel %s, callback error: %s", id, err)
			return fmt.Errorf("channel %s, callback error: %w", req.sl.Channel, err)
		}

		if !resp.HasMore {
			slog.DebugContext(ctx, "server reported channel done", "channel_id", req.sl.String())
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}

	return nil
}

// thread fetches the whole thread identified by SlackLink, calling callback
// function fn for each slice received. the callback function will be called if
// there's no messages in the thread, and should handle as it sees fit.
func (cs *Stream) thread(ctx context.Context, req request, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "thread")
	defer task.End()

	if !req.sl.IsThread() {
		return fmt.Errorf("not a thread: %s", req.sl)
	}

	slog.DebugContext(ctx, "- getting thread", "slack_link", req.sl)

	var cursor string
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
			var apiErr error
			msgs, hasmore, cursor, apiErr = cs.client.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
				ChannelID: req.sl.Channel,
				Timestamp: req.sl.ThreadTS,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Replies,
				Oldest:    structures.FormatSlackTS(structures.NVLTime(req.Oldest, cs.oldest)),
				Latest:    structures.FormatSlackTS(structures.NVLTime(req.Latest, cs.latest)),
				Inclusive: cs.inclusive,
			})
			if apiErr == nil && len(msgs) == 0 {
				slog.DebugContext(ctx, "  - no messages returned by the API, requesting a retry", "slack_link", req.sl)
				// no messages returned by the API, but no error either, let's ask
				// nicely to retry, maybe Slack is having a bad day.
				return network.ErrRetryPlease
			}
			return apiErr
		}); err != nil {
			return err
		}

		r := trace.StartRegion(ctx, "thread_callback")
		err := callback(msgs, !hasmore)
		r.End()
		if err != nil {
			return err
		}

		if !hasmore {
			break
		}
	}
	return nil
}

// procChanMsg processes the message slice mm, for each threaded message, it
// sends the thread request on threadC.  It returns thread count in the mm and
// error if any.
func procChanMsg(ctx context.Context, proc processor.Conversations, threadC chan<- request, channel *slack.Channel, isLast bool, mm []slack.Message) (int, error) {
	trs := make([]request, 0, len(mm))
	for i := range mm {
		// collecting threads to get their count.  But we don't start
		// processing them yet, before we send the messages with the number of
		// "expected" threads to processor, to ensure that processor will
		// start processing the channel and will have the initial reference
		// count, if it needs it.
		if mm[i].ThreadTimestamp != "" && mm[i].SubType != structures.SubTypeThreadBroadcast && mm[i].LatestReply != structures.LatestReplyNoReplies {
			slog.DebugContext(ctx, "- message", "i", i, "thread", mm[i].Timestamp, "thread_ts", mm[i].ThreadTimestamp, "channel_id", channel.ID, "is_last", isLast, "msg_count", len(mm))
			trs = append(trs, request{
				sl: &structures.SlackLink{
					Channel:  channel.ID,
					ThreadTS: mm[i].ThreadTimestamp,
				},
			})
		}
		if err := procFiles(ctx, proc, channel, mm[i]); err != nil {
			return len(trs), err
		}
	}
	if err := proc.Messages(ctx, channel.ID, len(trs), isLast, mm); err != nil {
		if len(mm) == 0 {
			return 0, fmt.Errorf("channel %s: failed to process empty message chunk: %w", channel.ID, err)
		}
		return 0, fmt.Errorf("channel %s: failed to process message chunk starting with id=%s (size=%d): %w", channel.ID, mm[0].Timestamp, len(mm), err)
	}
	for _, tr := range trs {
		threadC <- tr
	}
	return len(trs), nil
}

func procThreadMsg(ctx context.Context, proc processor.Conversations, channel *slack.Channel, threadTS string, threadOnly bool, isLast bool, msgs []slack.Message) error {
	lg := slog.With("channel_id", channel.ID, "thread_ts", threadTS, "is_last", isLast, "msg_count", len(msgs))
	if len(msgs) == 0 {
		lg.Debug("empty thread messages, ignoring")
		return nil
	}
	// slack returns the thread starter as the first message with every
	// call, so we use it as a head message.
	parent, rest := msgs[0], []slack.Message{}
	if len(msgs) > 1 {
		rest = msgs[1:]
	}
	// extract files from thread messages
	if err := procFiles(ctx, proc, channel, rest...); err != nil {
		return err
	}
	if err := proc.ThreadMessages(ctx, channel.ID, parent, threadOnly, isLast, msgs); err != nil {
		return fmt.Errorf("failed to process thread message id=%s, thread_ts=%s: %w", parent.Timestamp, threadTS, err)
	}
	return nil
}

// procFiles processes the files in slice of Messages msgs.
func procFiles(ctx context.Context, proc processor.Filer, channel *slack.Channel, msgs ...slack.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	for _, m := range msgs {
		if len(m.Files) > 0 {
			if err := proc.Files(ctx, channel, m, m.Files); err != nil {
				return err
			}
		}
	}
	return nil
}

// procChannelInfo fetches the channel info and passes it to the processor.
func (cs *Stream) procChannelInfo(ctx context.Context, proc processor.ChannelInformer, channelID string, threadTS string) (*slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "channelInfo")
	defer task.End()

	trace.Logf(ctx, "channel_id", "%s, threadTS=%q", channelID, threadTS)

	// to avoid fetching the same channel info multiple times, we cache it.
	var info *slack.Channel
	if info = cs.chanCache.get(channelID); info == nil {
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func(ctx context.Context) error {
			var err error
			info, err = cs.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
				ChannelID:         channelID,
				IncludeLocale:     true,
				IncludeNumMembers: true,
			})
			if err != nil {
				if structures.IsSlackResponseError(err, errChanNotFound.Error()) {
					return errChanNotFound
				}
				return fmt.Errorf("error getting channel information: %w", err)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("api error: %s: %w", channelID, err)
		}
		cs.chanCache.set(channelID, info)
	}
	if err := proc.ChannelInfo(ctx, info, threadTS); err != nil {
		return nil, err
	}

	return info, nil
}

var errChanNotFound = errors.New("channel_not_found")

func (cs *Stream) procChannelUsers(ctx context.Context, proc processor.ChannelInformer, channelID, threadTS string) ([]string, error) {
	var users []string

	var cursor string
	for {
		var u []string
		var next string
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier4.Retries, func(ctx context.Context) error {
			var err error
			u, next, err = cs.client.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{
				ChannelID: channelID,
				Cursor:    cursor,
			})
			return err
		}); err != nil {
			if structures.IsSlackResponseError(err, errChanNotFound.Error()) {
				return nil, errChanNotFound
			}
			return nil, fmt.Errorf("error getting conversation users: %w", err)
		}
		if len(u) == 0 && next == "" {
			break
		}
		if err := proc.ChannelUsers(ctx, channelID, threadTS, u); err != nil {
			return nil, err
		}
		users = append(users, u...)
		if next == "" {
			break
		}
		cursor = next
	}

	return users, nil
}

// procChannelInfoWithUsers returns the slack channel with members populated from
// another api.
func (cs *Stream) procChannelInfoWithUsers(ctx context.Context, proc processor.ChannelInformer, channelID, threadTS string) (*slack.Channel, error) {
	eg, ctx := errgroup.WithContext(ctx)

	chC := make(chan slack.Channel, 1)
	eg.Go(func() error {
		defer close(chC)
		ch, err := cs.procChannelInfo(ctx, proc, channelID, threadTS)
		if err != nil {
			return err
		}
		chC <- *ch
		return nil
	})

	uC := make(chan []string, 1)
	eg.Go(func() error {
		defer close(uC)
		m, err := cs.procChannelUsers(ctx, proc, channelID, threadTS)
		if err != nil {
			return err
		}
		uC <- m
		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	ch := <-chC
	ch.Members = <-uC
	return &ch, nil
}
