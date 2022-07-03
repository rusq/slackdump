package slackdump

import (
	"context"
	"time"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

// DumpAllURL dumps messages from the slackURL.
//
// Deprecated: Use DumpAll, this function will be removed in v3.
func (sd *Session) DumpAllURL(ctx context.Context, slackURL string) (*types.Conversation, error) {
	return sd.Dump(ctx, slackURL, time.Time{}, time.Time{})
}

// DumpURL acts like DumpAllURL but allows to specify oldest and latest
// timestamps to define a window within which the messages should be retrieved,
// also one can provide process functions.
//
// Deprecated: Use Dump, this function will be removed in v3.
func (sd *Session) DumpURL(ctx context.Context, slackURL string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	return sd.Dump(ctx, slackURL, oldest, latest, processFn...)
}

// DumpAllMessages fetches messages from the conversation identified by channelID.
//
// Deprecated: Use DumpAll, this function will be removed in v3.
func (sd *Session) DumpAllMessages(ctx context.Context, channelID string) (*types.Conversation, error) {
	return sd.Dump(ctx, channelID, time.Time{}, time.Time{})
}

// DumpMessages dumps messages in the given timeframe between oldest
// and latest.  If oldest or latest are zero time, they will not be accounted
// for.  Having both oldest and latest as Zero-time, will make this function
// behave similar to DumpMessages.  ProcessFn is a slice of post-processing functions
// that will be called for each message chunk downloaded from the Slack API.
//
// Deprecated: Use Dump, this function will be unexported in v3.
func (sd *Session) DumpMessages(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	return sd.Dump(ctx, channelID, oldest, latest, processFn...)
}

// DumpMessagesRaw dumps all messages, but does not account for any options
// defined, such as DumpFiles, instead, the caller must hassle about any
// processFns they want to apply.
//
// Deprecated: Use DumpRaw, this function will be unexported in v3.
func (sd *Session) DumpMessagesRaw(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*types.Conversation, error) {
	return sd.DumpRaw(ctx, channelID, oldest, latest, processFn...)
}

// DumpThread dumps a single thread identified by (channelID, threadTS).
// Optionally one can provide a number of processFn that will be applied to each
// chunk of messages returned by a one API call.
//
// Deprecated: Use Dump, this function will be unexported in v3.
func (sd *Session) DumpThread(
	ctx context.Context,
	channelID,
	threadTS string,
	oldest, latest time.Time,
	processFn ...ProcessFunc,
) (*types.Conversation, error) {
	sl := structures.SlackLink{Channel: channelID, ThreadTS: threadTS}
	return sd.Dump(ctx, sl.String(), oldest, latest, processFn...)
}
