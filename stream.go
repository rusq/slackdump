package slackdump

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/internal/event/processor"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

type channelStream struct {
	oldest, latest time.Time
	client         clienter
	limits         rateLimits
}

type rateLimits struct {
	channels *rate.Limiter
	threads  *rate.Limiter
	tier     *Limits
}

func newChannelStream(cl clienter, limits *Limits, oldest, latest time.Time) *channelStream {
	cs := &channelStream{
		oldest: oldest,
		latest: latest,
		client: cl,
		limits: rateLimits{
			channels: network.NewLimiter(network.Tier3, limits.Tier3.Burst, int(limits.Tier3.Boost)),
			threads:  network.NewLimiter(network.Tier3, limits.Tier3.Burst, int(limits.Tier3.Boost)),
			tier:     limits,
		},
	}
	return cs
}

func (cs *channelStream) Stream(ctx context.Context, link string, proc processor.Processor) error {
	ctx, task := trace.NewTask(ctx, "Stream")
	defer task.End()

	sl, err := structures.ParseLink(link)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return errors.New("invalid slack link: " + link)
	}
	if sl.IsThread() {
		if err := cs.thread(ctx, sl.Channel, sl.ThreadTS, proc); err != nil {
			return err
		}
	} else {
		if err := cs.channel(ctx, sl.Channel, proc); err != nil {
			return err
		}
	}
	return nil
}

func (cs *channelStream) channel(ctx context.Context, id string, proc processor.Processor) error {
	ctx, task := trace.NewTask(ctx, "channel")
	defer task.End()

	cursor := ""
	for {
		var (
			resp *slack.GetConversationHistoryResponse
		)
		if err := network.WithRetry(ctx, cs.limits.channels, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			rgn := trace.StartRegion(ctx, "GetConversationHistoryContext")
			resp, apiErr = cs.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
				ChannelID: id,
				Cursor:    cursor,
				Limit:     cs.limits.tier.Request.Conversations,
				Oldest:    structures.FormatSlackTS(cs.oldest),
				Latest:    structures.FormatSlackTS(cs.latest),
				Inclusive: true,
			})
			rgn.End()
			return apiErr
		}); err != nil {
			return err
		}
		if !resp.Ok {
			trace.Logf(ctx, "error", "not ok, api error=%s", resp.Error)
			return fmt.Errorf("response not ok, slack error: %s", resp.Error)
		}
		if err := proc.Messages(id, resp.Messages); err != nil {
			return fmt.Errorf("failed to process message chunk starting with id=%s (size=%d): %w", resp.Messages[0].Msg.ClientMsgID, len(resp.Messages), err)
		}
		for i := range resp.Messages {
			idx := i
			if resp.Messages[idx].Msg.ThreadTimestamp != "" && resp.Messages[idx].Msg.SubType != "thread_broadcast" {
				dlog.Debugf("- message #%d/thread: id=%s, thread_ts=%s, cursor=%s", i, resp.Messages[idx].ClientMsgID, resp.Messages[idx].Msg.ThreadTimestamp, cursor)
				if err := cs.thread(ctx, id, resp.Messages[idx].Msg.ThreadTimestamp, proc); err != nil {
					return err
				}
			}
			if resp.Messages[idx].Files != nil && len(resp.Messages[idx].Files) > 0 {
				if err := proc.Files(id, resp.Messages[idx], false, resp.Messages[idx].Files); err != nil {
					return err
				}
			}
		}
		if !resp.HasMore {
			break
		}
		cursor = resp.ResponseMetaData.NextCursor
	}
	return nil
}

func (cs *channelStream) thread(ctx context.Context, id string, threadTS string, proc processor.Processor) error {
	cursor := ""
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			dlog.Debugf("- getting: thread: id=%s, thread_ts=%s, cursor=%s", id, threadTS, cursor)
			msgs, hasmore, cursor, apiErr = cs.client.GetConversationRepliesContext(ctx, &slack.GetConversationRepliesParameters{
				ChannelID: id,
				Timestamp: threadTS,
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

		// slack returns the thread starter as the first message with every
		// call so we use it as a parent message.
		if err := proc.ThreadMessages(id, msgs[0], msgs[1:]); err != nil {
			return fmt.Errorf("failed to process message id=%s, thread_ts=%s: %w", msgs[0].Msg.ClientMsgID, threadTS, err)
		}
		// extract files from thread messages
		for i := range msgs[1:] {
			idx := i
			if msgs[idx].Files != nil && len(msgs[idx].Files) > 0 {
				if err := proc.Files(id, msgs[idx], true, msgs[idx].Files); err != nil {
					return err
				}
			}
		}
		if !hasmore {
			break
		}
	}
	return nil
}
