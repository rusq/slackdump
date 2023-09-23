package slackdump

import (
	"context"
	"fmt"
	"runtime/trace"
	"time"

	"errors"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

type threadFunc func(ctx context.Context, l *rate.Limiter, channelID string, threadTS string, oldest, latest time.Time, processFn ...ProcessFunc) ([]types.Message, error)

// dumpThreadAsConversation dumps a single thread identified by (channelID,
// threadTS). Optionally one can provide a number of processFn that will be
// applied to each chunk of messages returned by a one API call.
func (sd *Session) dumpThreadAsConversation(
	ctx context.Context,
	sl structures.SlackLink,
	oldest, latest time.Time,
	processFn ...ProcessFunc,
) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "DumpThread")
	defer task.End()

	if !(sl.IsValid() && sl.IsThread()) {
		return nil, errors.New("internal error: channelID or threadTS are empty")
	}

	trace.Logf(ctx, "info", "channelID: %q, threadTS: %q", sl.Channel, sl.ThreadTS)

	threadMsgs, err := sd.dumpThread(ctx, sd.limiter(network.Tier3), sl.Channel, sl.ThreadTS, oldest, latest, processFn...)
	if err != nil {
		return nil, err
	}

	types.SortMessages(threadMsgs)

	name, err := sd.getChannelName(ctx, sd.limiter(network.Tier3), sl.Channel)
	if err != nil {
		return nil, err
	}

	return &types.Conversation{
		Name:     name,
		Messages: threadMsgs,
		ID:       sl.Channel,
		ThreadTS: sl.ThreadTS,
	}, nil
}

// populateThreads scans the message slice for threads, if it discovers the
// message with ThreadTimestamp, it calls the dumpFn on it. dumpFn should return
// the messages from the thread. Returns the count of messages that contained
// threads.  msgs is being updated with discovered messages.
//
// ref: https://api.slack.com/messaging/retrieving
func (*Session) populateThreads(
	ctx context.Context,
	l *rate.Limiter,
	msgs []types.Message,
	channelID string,
	oldest, latest time.Time,
	dumpFn threadFunc,
) (int, error) {
	total := 0
	for i := range msgs {
		if msgs[i].ThreadTimestamp == "" || msgs[i].SubType == "thread_broadcast" {
			continue
		}
		threadMsgs, err := dumpFn(ctx, l, channelID, msgs[i].ThreadTimestamp, oldest, latest)
		if err != nil {
			return total, err
		}
		if len(threadMsgs) == 0 {
			trace.Log(ctx, "warn", "a very strange situation right here, no error, and no messages. testing?")
			continue
		}
		msgs[i].ThreadReplies = threadMsgs[1:] // the first message returned by conversation.history is the message that started thread, so skipping it.
		total++
	}
	return total, nil
}

// dumpThread retrieves all messages in the thread and returns them as a slice
// of messages.
func (sd *Session) dumpThread(
	ctx context.Context,
	l *rate.Limiter,
	channelID string,
	threadTS string,
	oldest, latest time.Time,
	processFn ...ProcessFunc,
) ([]types.Message, error) {
	var (
		thread     []types.Message
		cursor     string
		fetchStart = time.Now()
	)
	for i := 0; ; i++ {
		var (
			msgs       []slack.Message
			hasmore    bool
			nextCursor string
		)
		reqStart := time.Now()
		if err := network.WithRetry(ctx, l, sd.options.Tier3Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationRepliesContext", func() {
				msgs, hasmore, nextCursor, err = sd.client.GetConversationRepliesContext(
					ctx,
					&slack.GetConversationRepliesParameters{
						ChannelID: channelID,
						Cursor:    cursor,
						Timestamp: threadTS,
						Limit:     sd.options.RepliesPerReq,
						Oldest:    structures.FormatSlackTS(oldest),
						Latest:    structures.FormatSlackTS(latest),
						Inclusive: true,
					},
				)
			})
			if err != nil {
				return fmt.Errorf("failed to dump channel:thread %s:%s: %w", channelID, threadTS, err)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		// slack api returns the first message of a thread with every api call:
		// strip the first message if i > 0 to avoid dupes
		if 0 < i && 1 < len(msgs) {
			msgs = msgs[1:]
		}
		thread = append(thread, types.ConvertMsgs(msgs)...)

		prs, err := runProcessFuncs(thread, channelID, processFn...)
		if err != nil {
			return nil, err
		}

		sd.l().Printf("  thread request #%5d, fetched: %4d, total: %8d, process results: %s (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i+1, len(msgs), len(thread),
			prs,
			float64(len(msgs))/time.Since(reqStart).Seconds(),
			float64(len(thread))/time.Since(fetchStart).Seconds(),
		)

		if !hasmore {
			sd.l().Printf("  thread fetch complete, total: %d", len(thread))
			break
		}
		cursor = nextCursor
	}
	return thread, nil
}
