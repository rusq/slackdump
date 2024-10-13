package stream

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"sync"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/processor"
	"golang.org/x/sync/errgroup"
)

// SyncConversations fetches the conversations from the link which can be a
// channelID, channel URL, thread URL or a link in Slackdump format.
func (cs *Stream) SyncConversations(ctx context.Context, proc processor.Conversations, link ...string) error {
	lg := logger.FromContext(ctx)
	return cs.ConversationsCB(ctx, proc, link, func(sr Result) error {
		lg.Debugf("stream: finished processing: %s", sr)
		return nil
	})
}

func (cs *Stream) ConversationsCB(ctx context.Context, proc processor.Conversations, link []string, cb func(Result) error) error {
	ctx, task := trace.NewTask(ctx, "channelStream.Conversations")
	defer task.End()

	lg := logger.FromContext(ctx)
	cs.resultFn = append(cs.resultFn, cb)

	linkC := make(chan string, 1)
	go func() {
		defer close(linkC)
		for _, l := range link {
			linkC <- l
		}
		lg.Debugf("stream: sent %d links", len(link))
	}()

	if err := cs.Conversations(ctx, proc, linkC); err != nil {
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
func (cs *Stream) Conversations(ctx context.Context, proc processor.Conversations, links <-chan string) error {
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
					resultsC <- Result{Type: RTMain, Err: ctx.Err()}
					return
				case link, more := <-links:
					if !more {
						return
					}
					if err := processLink(chansC, threadsC, link); err != nil {
						resultsC <- Result{Type: RTMain, Err: fmt.Errorf("link error: %q: %w", link, err)}
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
			return err
		}
		for _, fn := range cs.resultFn {
			if err := fn(res); err != nil {
				return err
			}
		}
	}
	trace.Log(ctx, "info", "complete")
	return nil
}

// processLink parses the link and sends it to the appropriate output channel.
func processLink(chans chan<- request, threads chan<- request, link string) error {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return fmt.Errorf("invalid slack link: %s", link)
	}
	if sl.IsThread() {
		threads <- request{sl: &sl, threadOnly: true}
	} else {
		chans <- request{sl: &sl}
	}
	return nil
}

type request struct {
	sl *structures.SlackLink
	// threadOnly indicates that this is the thread directly requested by the
	// user, and not a thread that was found in the channel.
	threadOnly bool
}

func (we *Result) Error() string {
	return fmt.Sprintf("%s channel %s: %v", we.Type, structures.SlackLink{Channel: we.ChannelID, ThreadTS: we.ThreadTS}, we.Err)
}

func (we *Result) Unwrap() error {
	return we.Err
}

func (cs *Stream) channel(ctx context.Context, id string, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "channel")
	defer task.End()

	lg := logger.FromContext(ctx)

	cursor := ""
	for {
		var resp *slack.GetConversationHistoryResponse
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			r := trace.StartRegion(ctx, "GetConversationHistoryContext")
			defer r.End()
			resp, apiErr = cs.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
				ChannelID: id,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Conversations,
				Oldest:    structures.FormatSlackTS(cs.oldest),
				Latest:    structures.FormatSlackTS(cs.latest),
				Inclusive: true,
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
			return fmt.Errorf("channel %s, callback error: %w", id, err)
		}

		if !resp.HasMore {
			lg.Debugf("server reported channel %s done", id)
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}
	return nil
}

// thread fetches the whole thread identified by SlackLink, calling callback
// function fn for each slice received.
func (cs *Stream) thread(ctx context.Context, sl *structures.SlackLink, callback func(mm []slack.Message, isLast bool) error) error {
	ctx, task := trace.NewTask(ctx, "thread")
	defer task.End()

	if !sl.IsThread() {
		return fmt.Errorf("not a thread: %s", sl)
	}

	lg := logger.FromContext(ctx)
	lg.Debugf("- getting: %s", sl)

	var cursor string
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			msgs, hasmore, cursor, apiErr = cs.client.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
				ChannelID: sl.Channel,
				Timestamp: sl.ThreadTS,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Replies,
				Oldest:    structures.FormatSlackTS(cs.oldest),
				Latest:    structures.FormatSlackTS(cs.latest),
				Inclusive: true,
			})
			return apiErr
		}); err != nil {
			return err
		}

		// got just the leader message, no replies
		if len(msgs) <= 1 {
			return nil
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
	lg := logger.FromContext(ctx)

	var trs = make([]request, 0, len(mm))
	for i := range mm {
		// collecting threads to get their count.  But we don't start
		// processing them yet, before we send the messages with the number of
		// "expected" threads to processor, to ensure that processor will
		// start processing the channel and will have the initial reference
		// count, if it needs it.
		if mm[i].Msg.ThreadTimestamp != "" && mm[i].Msg.SubType != structures.SubTypeThreadBroadcast && mm[i].LatestReply != structures.LatestReplyNoReplies {
			lg.Debugf("- message #%d/channel=%s,thread: id=%s, thread_ts=%s", i, channel.ID, mm[i].Timestamp, mm[i].Msg.ThreadTimestamp)
			trs = append(trs, request{
				sl: &structures.SlackLink{
					Channel:  channel.ID,
					ThreadTS: mm[i].Msg.ThreadTimestamp,
				},
			})
		}
		if err := procFiles(ctx, proc, channel, mm[i]); err != nil {
			return len(trs), err
		}
	}
	if err := proc.Messages(ctx, channel.ID, len(trs), isLast, mm); err != nil {
		return 0, fmt.Errorf("failed to process message chunk starting with id=%s (size=%d): %w", mm[0].Msg.Timestamp, len(mm), err)
	}
	for _, tr := range trs {
		threadC <- tr
	}
	return len(trs), nil
}

func procThreadMsg(ctx context.Context, proc processor.Conversations, channel *slack.Channel, threadTS string, threadOnly bool, isLast bool, msgs []slack.Message) error {
	// extract files from thread messages
	if len(msgs) == 0 {
		return errors.New("empty messages slice")
	}
	if err := procFiles(ctx, proc, channel, msgs[1:]...); err != nil {
		return err
	}
	// slack returns the thread starter as the first message with every
	// call, so we use it as a parent message.
	if err := proc.ThreadMessages(ctx, channel.ID, msgs[0], threadOnly, isLast, msgs[1:]); err != nil {
		return fmt.Errorf("failed to process thread message id=%s, thread_ts=%s: %w", msgs[0].Msg.Timestamp, threadTS, err)
	}
	return nil
}

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
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
			var err error
			info, err = cs.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
				ChannelID:         channelID,
				IncludeLocale:     true,
				IncludeNumMembers: true,
			})
			if err != nil {
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

func (cs *Stream) procChannelUsers(ctx context.Context, proc processor.ChannelInformer, channelID, threadTS string) ([]string, error) {
	var users []string

	var cursor string
	for {
		var u []string
		var next string
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier4.Retries, func() error {
			var err error
			u, next, err = cs.client.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{
				ChannelID: channelID,
				Cursor:    cursor,
			})
			return err
		}); err != nil {
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
	var eg errgroup.Group

	var chC = make(chan slack.Channel, 1)
	eg.Go(func() error {
		defer close(chC)
		ch, err := cs.procChannelInfo(ctx, proc, channelID, threadTS)
		if err != nil {
			return err
		}
		chC <- *ch
		return nil
	})

	var uC = make(chan []string, 1)
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
