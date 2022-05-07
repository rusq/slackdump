package slackdump

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/network"
)

type threadFunc func(ctx context.Context, l *rate.Limiter, channelID string, threadTS string, processFn ...ProcessFunc) ([]Message, error)

// DumpThread dumps a single thread identified by (channelID, threadTS).
// Optionally one can provide a number of processFn that will be applied to each
// chunk of messages returned by a one API call.
func (sd *SlackDumper) DumpThread(ctx context.Context, channelID, threadTS string, processFn ...ProcessFunc) (*Conversation, error) {
	ctx, task := trace.NewTask(ctx, "DumpThread")
	defer task.End()

	if threadTS == "" || channelID == "" {
		return nil, errors.New("internal error: channelID or threadTS are empty")
	}

	trace.Logf(ctx, "info", "channelID: %q, threadTS: %q", channelID, threadTS)

	if sd.options.DumpFiles {
		fn, cancelFn, err := sd.newFileProcessFn(ctx, channelID, sd.limiter(network.NoTier))
		if err != nil {
			return nil, err
		}
		defer cancelFn()
		processFn = append(processFn, fn)
	}

	threadMsgs, err := sd.dumpThread(ctx, sd.limiter(network.Tier3), channelID, threadTS, processFn...)
	if err != nil {
		return nil, err
	}

	sortMessages(threadMsgs)

	name, err := sd.getChannelName(ctx, sd.limiter(network.Tier3), channelID)
	if err != nil {
		return nil, err
	}

	return &Conversation{
		Name:     name,
		Messages: threadMsgs,
		ID:       channelID,
		ThreadTS: threadTS,
	}, nil
}

// populateThreads scans the message slice for threads, if it discovers the
// message with ThreadTimestamp, it calls the dumpFn on it. dumpFn should return
// the messages from the thread. Returns the count of messages that contained
// threads.  msgs is being updated with discovered messages.
//
// ref: https://api.slack.com/messaging/retrieving
func (*SlackDumper) populateThreads(ctx context.Context, l *rate.Limiter, msgs []Message, channelID string, dumpFn threadFunc) (int, error) {
	total := 0
	for i := range msgs {
		if msgs[i].ThreadTimestamp == "" {
			continue
		}
		threadMsgs, err := dumpFn(ctx, l, channelID, msgs[i].ThreadTimestamp)
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
func (sd *SlackDumper) dumpThread(ctx context.Context, l *rate.Limiter, channelID string, threadTS string, processFn ...ProcessFunc) ([]Message, error) {
	var (
		thread     []Message
		cursor     string
		fetchStart = time.Now()
	)
	for i := 1; ; i++ {
		var (
			msgs       []slack.Message
			hasmore    bool
			nextCursor string
		)
		reqStart := time.Now()
		if err := withRetry(ctx, l, sd.options.Tier3Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationRepliesContext", func() {
				msgs, hasmore, nextCursor, err = sd.client.GetConversationRepliesContext(
					ctx,
					&slack.GetConversationRepliesParameters{ChannelID: channelID, Timestamp: threadTS, Cursor: cursor},
				)
			})
			return errors.WithStack(err)
		}); err != nil {
			return nil, err
		}

		thread = append(thread, sd.convertMsgs(msgs)...)

		prs, err := runProcessFuncs(thread, channelID, processFn...)
		if err != nil {
			return nil, err
		}

		dlog.Printf("  thread request #%5d, fetched: %4d, total: %8d, process results: %s (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i, len(msgs), len(thread),
			prs,
			float64(len(msgs))/float64(time.Since(reqStart).Seconds()),
			float64(len(thread))/float64(time.Since(fetchStart).Seconds()),
		)

		if !hasmore {
			dlog.Printf("  thread fetch complete, total: %d", len(thread))
			break
		}
		cursor = nextCursor
	}
	return thread, nil
}
