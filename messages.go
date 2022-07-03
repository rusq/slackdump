package slackdump

// In this file: messages related code.

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
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
func (sd *Session) Dump(ctx context.Context, link string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return nil, err
	}
	if sd.options.DumpFiles {
		fn, cancelFn, err := sd.newFileProcessFn(ctx, sl.Channel, sd.limiter(network.NoTier))
		if err != nil {
			return nil, err
		}
		defer cancelFn()
		processFn = append(processFn, fn)
	}

	return sd.dump(ctx, sl, oldest, latest, processFn...)
}

// DumpAll dumps all messages.  See description of Dump for what can be provided
// in link.
func (sd *Session) DumpAll(ctx context.Context, link string) (*types.Conversation, error) {
	return sd.Dump(ctx, link, time.Time{}, time.Time{})
}

// DumpRaw dumps all messages, but does not account for any options
// defined, such as DumpFiles, instead, the caller must hassle about any
// processFns they want to apply.
func (sd *Session) DumpRaw(ctx context.Context, link string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	sl, err := structures.ParseLink(link)
	if err != nil {
		return nil, err
	}
	return sd.dump(ctx, sl, oldest, latest, processFn...)
}

func (sd *Session) dump(ctx context.Context, sl structures.SlackLink, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dump")
	defer task.End()
	trace.Logf(ctx, "info", "sl: %q", sl)
	if !sl.IsValid() {
		return nil, errors.New("invalid link")
	}

	if sl.IsThread() {
		return sd.dumpThreadAsConversation(ctx, sl, oldest, latest, processFn...)
	} else {
		return sd.dumpChannel(ctx, sl.Channel, oldest, latest, processFn...)
	}
}

// dumpChannel fetches messages from the conversation identified by channelID.
// processFn will be called on each batch of messages returned from API.
func (sd *Session) dumpChannel(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dumpMessages")
	defer task.End()

	if channelID == "" {
		return nil, errors.New("channelID is empty")
	}

	trace.Logf(ctx, "info", "channelID: %q, oldest: %s, latest: %s", channelID, oldest, latest)

	var (
		// slack rate limits are per method, so we're safe to use different limiters for different mehtods.
		convLimiter   = sd.limiter(network.Tier3)
		threadLimiter = sd.limiter(network.Tier3)
	)

	// add thread dumper.  It should go first, because it populates message
	// chunk with thread messages.
	pfns := append([]ProcessFunc{sd.newThreadProcessFn(ctx, threadLimiter, oldest, latest)}, processFn...)

	var (
		messages   []types.Message
		cursor     string
		fetchStart = time.Now()
	)
	for i := 1; ; i++ {
		var (
			resp *slack.GetConversationHistoryResponse
		)
		reqStart := time.Now()
		if err := withRetry(ctx, convLimiter, sd.options.Tier3Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationHistoryContext", func() {
				resp, err = sd.client.GetConversationHistoryContext(ctx, &slack.GetConversationHistoryParameters{
					ChannelID: channelID,
					Cursor:    cursor,
					Limit:     sd.options.ConversationsPerReq,
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

		sd.l().Printf("messages request #%5d, fetched: %4d (%s), total: %8d (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i, len(resp.Messages), results, len(messages),
			float64(len(resp.Messages))/float64(time.Since(reqStart).Seconds()),
			float64(len(messages))/float64(time.Since(fetchStart).Seconds()),
		)

		if !resp.HasMore {
			sd.l().Printf("messages fetch complete, total: %d", len(messages))
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	types.SortMessages(messages)

	name, err := sd.getChannelName(ctx, sd.limiter(network.Tier3), channelID)
	if err != nil {
		return nil, err
	}

	return &types.Conversation{Name: name, Messages: messages, ID: channelID}, nil
}

func (sd *Session) getChannelName(ctx context.Context, l *rate.Limiter, channelID string) (string, error) {
	// get channel name
	var ci *slack.Channel
	if err := withRetry(ctx, l, sd.options.Tier3Retries, func() error {
		var err error
		ci, err = sd.client.GetConversationInfoContext(ctx, channelID, false)
		return err
	}); err != nil {
		return "", err
	}
	return ci.NameNormalized, nil
}
