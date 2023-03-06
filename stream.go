package slackdump

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/chunk/processor"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
)

type stream struct {
	oldest, latest time.Time
	client         clienter
	limits         rateLimits
}

type rateLimits struct {
	channels *rate.Limiter
	threads  *rate.Limiter
	users    *rate.Limiter
	tier     *Limits
}

func limits(l *Limits) rateLimits {
	return rateLimits{
		channels: network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		threads:  network.NewLimiter(network.Tier3, l.Tier3.Burst, int(l.Tier3.Boost)),
		users:    network.NewLimiter(network.Tier2, l.Tier2.Burst, int(l.Tier2.Boost)),
		tier:     l,
	}
}

func newChannelStream(cl clienter, l *Limits, oldest, latest time.Time) *stream {
	cs := &stream{
		oldest: oldest,
		latest: latest,
		client: cl,
		limits: limits(l),
	}
	return cs
}

func (cs *stream) Conversations(ctx context.Context, link string, proc processor.Conversations) error {
	ctx, task := trace.NewTask(ctx, "channelStream.Conversations")
	defer task.End()

	sl, err := structures.ParseLink(link)
	if err != nil {
		return err
	}
	if !sl.IsValid() {
		return errors.New("invalid slack link: " + link)
	}
	if sl.IsThread() {
		// we need to fetch the channel info on this level, because
		// thread is also being called from the channel, and we don't
		// want to fetch it every time.
		if err := cs.channelInfo(ctx, sl.Channel, true, proc); err != nil {
			return err
		}
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

// channelInfo fetches the channel info and passes it to the processor.
func (cs *stream) channelInfo(ctx context.Context, channelID string, isThread bool, proc processor.Conversations) error {
	ctx, task := trace.NewTask(ctx, "channelInfo")
	defer task.End()

	info, err := cs.client.GetConversationInfoContext(ctx, &slack.GetConversationInfoInput{
		ChannelID: channelID,
	})
	if err != nil {
		return err
	}
	if err := proc.ChannelInfo(ctx, info, isThread); err != nil {
		return err
	}
	return nil
}

func (cs *stream) channel(ctx context.Context, id string, proc processor.Conversations) error {
	ctx, task := trace.NewTask(ctx, "channel")
	defer task.End()

	if err := cs.channelInfo(ctx, id, false, proc); err != nil {
		return err
	}
	cursor := ""
	for {
		var resp *slack.GetConversationHistoryResponse
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
		if err := proc.Messages(ctx, id, resp.Messages); err != nil {
			return fmt.Errorf("failed to process message chunk starting with id=%s (size=%d): %w", resp.Messages[0].Msg.ClientMsgID, len(resp.Messages), err)
		}
		for i := range resp.Messages {
			if resp.Messages[i].Msg.ThreadTimestamp != "" && resp.Messages[i].Msg.SubType != "thread_broadcast" {
				dlog.Debugf("- message #%d/thread: id=%s, thread_ts=%s, cursor=%s", i, resp.Messages[i].ClientMsgID, resp.Messages[i].Msg.ThreadTimestamp, cursor)
				if err := cs.thread(ctx, id, resp.Messages[i].Msg.ThreadTimestamp, proc); err != nil {
					return err
				}
			}
			if len(resp.Messages[i].Files) > 0 {
				if err := proc.Files(ctx, id, resp.Messages[i], false, resp.Messages[i].Files); err != nil {
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

func (cs *stream) thread(ctx context.Context, id string, threadTS string, proc processor.Conversations) error {
	ctx, task := trace.NewTask(ctx, "thread")
	defer task.End()

	lg := dlog.FromContext(ctx)
	lg.Debugf("- getting: thread: id=%s, thread_ts=%s", id, threadTS)

	var cursor string
	for {
		var (
			msgs    []slack.Message
			hasmore bool
		)
		if err := network.WithRetry(ctx, cs.limits.threads, cs.limits.tier.Tier3.Retries, func() error {
			var apiErr error
			lg.Debugln("-- cursor", cursor)
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
		if err := proc.ThreadMessages(ctx, id, msgs[0], msgs[1:]); err != nil {
			return fmt.Errorf("failed to process message id=%s, thread_ts=%s: %w", msgs[0].Msg.ClientMsgID, threadTS, err)
		}
		// extract files from thread messages
		for _, m := range msgs[1:] {
			if len(m.Files) > 0 {
				if err := proc.Files(ctx, id, m, true, m.Files); err != nil {
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

func (cs *stream) Users(ctx context.Context, proc processor.Users) error {
	ctx, task := trace.NewTask(ctx, "Users")
	defer task.End()

	ti, err := cs.client.GetTeamInfoContext(ctx)
	if err != nil {
		return err
	}
	if err := proc.TeamInfo(ctx, ti); err != nil {
		return err
	}

	p := cs.client.GetUsersPaginated()
	var apiErr error
	for apiErr == nil {
		if apiErr := network.WithRetry(ctx, cs.limits.users, cs.limits.tier.Tier2.Retries, func() error {
			var err error
			p, err = p.Next(ctx)
			return err
		}); apiErr != nil {
			return apiErr
		}
		if err := proc.Users(ctx, ti.ID, p.Users); err != nil {
			return err
		}
	}

	return p.Failure(apiErr)
}
