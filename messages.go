package slackdump

// In this file: messages related code.

import (
	"bufio"
	"context"
	"fmt"
	"html"
	"io"
	"runtime/trace"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

// time format for text output.
const textTimeFmt = "02/01/2006 15:04:05 Z0700"

const (
	// minMsgTimeApart defines the time interval in minutes to separate group
	// of messages from a single user in the conversation.  This increases the
	// readability of the text output.
	minMsgTimeApart = 2 * time.Minute
)

// Channel keeps the slice of messages.
//
// Deprecated: use Conversation instead.
type Channel = Conversation

type ProcessResult struct {
	Entity string
	Count  int
}

type ProcessResults []ProcessResult

func (pr ProcessResult) String() string {
	return fmt.Sprintf("%s: %d", pr.Entity, pr.Count)
}

func (prs ProcessResults) String() string {
	var results []string
	for _, res := range prs {
		results = append(results, res.String())
	}
	return strings.Join(results, ", ")
}

// Conversation keeps the slice of messages.
type Conversation struct {
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
	// ID is the channel ID.
	ID string `json:"channel_id"`
	// ThreadTS is a thread timestamp.  If it's not empty, it means that it's a
	// dump of a thread, not a channel.
	ThreadTS string `json:"thread_ts,omitempty"`
}

// Message is the internal representation of message with thread.
type Message struct {
	slack.Message
	ThreadReplies []Message `json:"slackdump_thread_replies,omitempty"`
}

func (m Message) Datetime() (time.Time, error) {
	return parseSlackTS(m.Timestamp)
}

// IsBotMessage returns true if the message is from a bot.
func (m Message) IsBotMessage() bool {
	return m.Msg.BotID != ""
}

func (m Message) IsThread() bool {
	return m.Msg.ThreadTimestamp != ""
}

// IsThreadChild will return true if the message is the parent message of a
// conversation (has more than 0 replies)
func (m Message) IsThreadParent() bool {
	return m.IsThread() && m.Msg.ReplyCount != 0
}

// IsThreadChild will return true if the message is the child message of a
// conversation.
func (m Message) IsThreadChild() bool {
	return m.IsThread() && m.Msg.ReplyCount == 0
}

func (c Conversation) String() string {
	if c.ThreadTS == "" {
		return c.ID
	}
	return c.ID + "-" + c.ThreadTS
}

func (c Conversation) IsThread() bool {
	return c.ThreadTS != ""
}

// ToText outputs Messages m to io.Writer w in text format.
func (c Conversation) ToText(w io.Writer, sd *SlackDumper) (err error) {
	buf := bufio.NewWriter(w)
	defer buf.Flush()

	return sd.generateText(w, c.Messages, "")
}

// DumpAllURL dumps messages from the slack URL, it supports conversations and
// individual threads.
func (sd *SlackDumper) DumpAllURL(ctx context.Context, slackURL string) (*Conversation, error) {
	return sd.dumpURL(ctx, slackURL, time.Time{}, time.Time{})
}

// DumpURL acts like DumpURL but allows to specify oldest and latest
// timestamps to define a window within which the messages should be retrieved.
func (sd *SlackDumper) DumpURL(ctx context.Context, slackURL string, oldest, latest time.Time, processFn ...ProcessFunc) (*Conversation, error) {
	return sd.dumpURL(ctx, slackURL, oldest, latest, processFn...)
}

func (sd *SlackDumper) dumpURL(ctx context.Context, slackURL string, oldest, latest time.Time, processFn ...ProcessFunc) (*Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dumpURL")
	defer task.End()

	trace.Logf(ctx, "info", "slackURL: %q", slackURL)

	ui, err := parseURL(slackURL)
	if err != nil {
		return nil, err
	}

	if ui.IsThread() {
		return sd.DumpThread(ctx, ui.Channel, ui.ThreadTS, processFn...)
	} else {
		return sd.DumpMessages(ctx, ui.Channel, oldest, latest, processFn...)
	}
}

// DumpAllMessages fetches messages from the conversation identified by channelID.
func (sd *SlackDumper) DumpAllMessages(ctx context.Context, channelID string) (*Conversation, error) {
	return sd.DumpMessages(ctx, channelID, time.Time{}, time.Time{})
}

// DumpMessages dumps messages in the given timeframe between oldest
// and latest.  If oldest or latest are zero time, they will not be accounted
// for.  Having both oldest and latest as Zero-time, will make this function
// behave similar to DumpMessages.  ProcessFn is a slice of post-processing functions
// that will be called for each message chunk downloaded from the Slack API.
func (sd *SlackDumper) DumpMessages(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*Conversation, error) {
	if sd.options.DumpFiles {
		fn, cancelFn, err := sd.newFileProcessFn(ctx, channelID, sd.limiter(noTier))
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
func (sd *SlackDumper) DumpMessagesRaw(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*Conversation, error) {
	return sd.dumpMessages(ctx, channelID, oldest, latest, processFn...)
}

// DumpMessages fetches messages from the conversation identified by channelID.
// processFn will be called on each batch of messages returned from API.
func (sd *SlackDumper) dumpMessages(ctx context.Context, channelID string, oldest, latest time.Time, processFn ...ProcessFunc) (*Conversation, error) {
	ctx, task := trace.NewTask(ctx, "dumpMessages")
	defer task.End()

	if channelID == "" {
		return nil, errors.New("channelID is empty")
	}

	trace.Logf(ctx, "info", "channelID: %q, oldest: %s, latest: %s", channelID, oldest, latest)

	var (
		// slack rate limits are per method, so we're safe to use different limiters for different mehtods.
		convLimiter   = sd.limiter(tier3)
		threadLimiter = sd.limiter(tier3)
	)

	// add thread dumper.  It should go first, because it populates message
	// chunk with thread messages.
	pfns := append([]ProcessFunc{sd.newThreadProcessFn(ctx, threadLimiter)}, processFn...)

	var (
		messages   []Message
		cursor     string
		fetchStart = time.Now()
	)
	for i := 1; ; i++ {
		var (
			resp   *slack.GetConversationHistoryResponse
			params = sd.convHistoryParams(channelID, cursor, oldest, latest)
		)
		reqStart := time.Now()
		if err := withRetry(ctx, convLimiter, sd.options.Tier3Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationHistoryContext", func() {
				resp, err = sd.client.GetConversationHistoryContext(ctx, params)
			})
			return errors.WithStack(err)
		}); err != nil {
			return nil, err
		}
		if !resp.Ok {
			trace.Logf(ctx, "error", "not ok, api error=%s", resp.Error)
			return nil, fmt.Errorf("response not ok, slack error: %s", resp.Error)
		}

		chunk := sd.convertMsgs(resp.Messages)
		// threads, err := sd.populateThreads(ctx, threadLimiter, chunk, channelID, sd.dumpThread)
		// if err != nil {
		// 	return nil, err
		// }

		results, err := runProcessFuncs(chunk, channelID, pfns...)
		if err != nil {
			return nil, err
		}

		messages = append(messages, chunk...)

		dlog.Printf("messages request #%5d, fetched: %4d (%s), total: %8d (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i, len(resp.Messages), results, len(messages),
			float64(len(resp.Messages))/float64(time.Since(reqStart).Seconds()),
			float64(len(messages))/float64(time.Since(fetchStart).Seconds()),
		)

		if !resp.HasMore {
			dlog.Printf("messages fetch complete, total: %d", len(messages))
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	sortMessages(messages)

	name, err := sd.getChannelName(ctx, sd.limiter(tier3), channelID)
	if err != nil {
		return nil, err
	}

	return &Conversation{Name: name, Messages: messages, ID: channelID}, nil
}

func (sd *SlackDumper) getChannelName(ctx context.Context, l *rate.Limiter, channelID string) (string, error) {
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

// convHistoryParams returns GetConversationHistoryParameters.
func (sd *SlackDumper) convHistoryParams(channelID, cursor string, oldest, latest time.Time) *slack.GetConversationHistoryParameters {
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Cursor:    cursor,
		Limit:     sd.options.ConversationsPerReq,
	}
	if !oldest.IsZero() {
		params.Oldest = formatSlackTS(oldest)
		params.Inclusive = true // make sure we include the messages at this exact TS
	}
	if !latest.IsZero() {
		params.Latest = formatSlackTS(latest)
		params.Inclusive = true
	}
	return params
}

func (sd *SlackDumper) generateText(w io.Writer, m []Message, prefix string) error {
	var (
		prevMsg  Message
		prevTime time.Time
	)
	for _, message := range m {
		t, err := parseSlackTS(message.Timestamp)
		if err != nil {
			return err
		}
		diff := t.Sub(prevTime)
		if prevMsg.User == message.User && diff < minMsgTimeApart {
			fmt.Fprintf(w, prefix+"%s\n", message.Text)
		} else {
			fmt.Fprintf(w, prefix+"\n"+prefix+"> %s [%s] @ %s:\n%s\n",
				sd.SenderName(&message), message.User,
				t.Format(textTimeFmt),
				prefix+html.UnescapeString(message.Text),
			)
		}
		if len(message.ThreadReplies) > 0 {
			if err := sd.generateText(w, message.ThreadReplies, "|   "); err != nil {
				return err
			}
		}
		prevMsg = message
		prevTime = t
	}
	return nil
}

// SenderName returns username for the message
func (sd *SlackDumper) SenderName(msg *Message) string {
	var userid string
	if msg.Comment != nil {
		userid = msg.Comment.User
	} else {
		userid = msg.User
	}

	if userid != "" {
		return sd.username(userid)
	}

	return ""
}

func sortMessages(msgs []Message) {
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Timestamp < msgs[j].Timestamp
	})
}

// convertMsgs converts a slice of slack.Message to []Message.
func (*SlackDumper) convertMsgs(sm []slack.Message) []Message {
	msgs := make([]Message, len(sm))
	for i := range sm {
		msgs[i].Message = sm[i]
	}
	return msgs
}
