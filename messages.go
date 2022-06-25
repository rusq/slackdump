package slackdump

// In this file: messages related code.

import (
	"context"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

// DumpAllURL dumps messages from the slack URL, it supports conversations and
// individual threads.
func (sd *Session) DumpAllURL(ctx context.Context, slackURL string) (*types.Conversation, error) {
	return sd.dumpURL(ctx, slackURL, time.Time{}, time.Time{})
}

// DumpURL acts like DumpURL but allows to specify oldest and latest
// timestamps to define a window within which the messages should be retrieved.
func (sd *Session) DumpURL(ctx context.Context, slackURL string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	return sd.dumpURL(ctx, slackURL, oldest, latest, processFn...)
}

func (sd *Session) dumpURL(ctx context.Context, slackURL string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dumpURL")
	defer task.End()

	trace.Logf(ctx, "info", "slackURL: %q", slackURL)

	ui, err := structures.ParseURL(slackURL)
	if err != nil {
		return nil, err
	}

	if ui.IsThread() {
		return sd.DumpThread(ctx, ui.Channel, ui.ThreadTS, oldest, latest, processFn...)
	} else {
		return sd.DumpMessages(ctx, ui.Channel, oldest, latest, processFn...)
	}
}

// DumpAllMessages fetches messages from the conversation identified by channelID.
func (sd *Session) DumpAllMessages(ctx context.Context, channelID string) (*types.Conversation, error) {
	return sd.DumpMessages(ctx, channelID, time.Time{}, time.Time{})
}

// DumpMessages dumps messages in the given timeframe between oldest
// and latest.  If oldest or latest are zero time, they will not be accounted
// for.  Having both oldest and latest as Zero-time, will make this function
// behave similar to DumpMessages.  ProcessFn is a slice of post-processing functions
// that will be called for each message chunk downloaded from the Slack API.
func (sd *Session) DumpMessages(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	if sd.options.DumpFiles {
		fn, cancelFn, err := sd.newFileProcessFn(ctx, channelID, sd.limiter(network.NoTier))
		if err != nil {
			return nil, err
		}
		defer cancelFn()
		processFn = append(processFn, fn)
	}

	return sd.dumpMessages(ctx, channelID, oldest, latest, processFn...)
}

// DumpMessagesRaw dumps all messages, but does not account for any options
// defined, such as DumpFiles, instead, the caller must hassle about any
// processFns they want to apply.
func (sd *Session) DumpMessagesRaw(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	return sd.dumpMessages(ctx, channelID, oldest, latest, processFn...)
}

// DumpMessages fetches messages from the conversation identified by channelID.
// processFn will be called on each batch of messages returned from API.
func (sd *Session) dumpMessages(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
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
			return errors.WithStack(err)
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
