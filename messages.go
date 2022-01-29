package slackdump

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"runtime/trace"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const (
	// minMsgTimeApart defines the time interval in minutes to separate group
	// of messages from a single user in the conversation.  This increases the
	// readability of the text output.
	minMsgTimeApart = 2
	// files channel buffer size. I don't know, i just like 20, doesn't really matter.
	filesCbufSz = 20
)

// Channel keeps the slice of messages.
//
// Deprecated: use Conversation instead.
type Channel = Conversation

// Conversation keeps the slice of messages.
type Conversation struct {
	Messages []Message `json:"messages"`
	// ID is the channel ID.
	ID string `json:"channel_id"`
	// ThreadTS is a thread timestamp.  If it's not empty, it means that it's a
	// dump of a thread, not a channel.
	ThreadTS string `json:"thread_ts,omitempty"`
}

func (c Conversation) String() string {
	if c.ThreadTS == "" {
		return c.ID
	}
	return c.ID + "-" + c.ThreadTS
}

// Message is the internal representation of message with thread.
type Message struct {
	slack.Message
	ThreadReplies []Message `json:"slackdump_thread_replies,omitempty"`
}

// ToText outputs Messages m to io.Writer w in text format.
func (m Conversation) ToText(sd *SlackDumper, w io.Writer) (err error) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	return generateText(w, sd, m.Messages, "")
}

// DumpURL dumps messages from the slack URL, it supports conversations and individual threads.
func (sd *SlackDumper) DumpURL(ctx context.Context, slackURL string) (*Conversation, error) {
	ui, err := ParseURL(slackURL)
	if err != nil {
		return nil, err
	}
	if !ui.IsValid() {
		return nil, fmt.Errorf("invalid URL: %q", slackURL)
	}

	if ui.IsThread() {
		return sd.DumpThread(ctx, ui.Channel, ui.Thread)
	} else {
		return sd.DumpMessages(ctx, ui.Channel)
	}
	// unreachable
}

// DumpMessages fetches messages from the conversation identified by channelID.
func (sd *SlackDumper) DumpMessages(ctx context.Context, channelID string) (*Conversation, error) {
	ctx, task := trace.NewTask(ctx, "DumpMessages")
	defer task.End()

	var (
		// slack rate limits are per method, so we're safe to use different limiters for different mehtods.
		convLimiter   = sd.limiter(tier3)
		threadLimiter = sd.limiter(tier3)
		dlLimiter     = sd.limiter(noTier) // go-slack/slack just sends the Post to the file endpoint, so this should work.
	)

	var filesC = make(chan *slack.File, filesCbufSz)
	dlDoneC, err := sd.newFileDownloader(ctx, dlLimiter, channelID, filesC)
	if err != nil {
		return nil, err
	}

	var (
		messages []Message
		cursor   string
	)
	for i := 1; ; i++ {
		var resp *slack.GetConversationHistoryResponse
		if err := withRetry(ctx, convLimiter, sd.options.conversationRetries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationHistoryContext", func() {
				resp, err = sd.client.GetConversationHistoryContext(
					ctx,
					&slack.GetConversationHistoryParameters{
						ChannelID: channelID,
						Cursor:    cursor,
						Limit:     sd.options.conversationsPerRequest,
					},
				)
			})
			return err
		}); err != nil {
			return nil, err
		}

		chunk := sd.convertMsgs(resp.Messages)
		threads, err := sd.populateThreads(ctx, threadLimiter, chunk, channelID)
		if err != nil {
			return nil, err
		}
		sd.pipeFiles(filesC, chunk)
		messages = append(messages, chunk...)

		dlog.Printf("request #%5d, fetched: %4d, (with threads: %4d) total: %8d\n",
			i, len(resp.Messages), threads, len(messages))

		if !resp.HasMore {
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	if sd.options.dumpfiles {
		close(filesC)
		<-dlDoneC
	}

	sortMessages(messages)

	return &Conversation{Messages: messages, ID: channelID}, nil
}

func generateText(w io.Writer, sd *SlackDumper, m []Message, prefix string) error {
	var (
		prevMsg  Message
		prevTime time.Time
	)
	for _, message := range m {
		t, err := fromSlackTime(message.Timestamp)
		if err != nil {
			return err
		}
		diff := t.Sub(prevTime)
		if prevMsg.User == message.User && diff.Minutes() < minMsgTimeApart {
			fmt.Fprintf(w, prefix+"%s\n", message.Text)
		} else {
			fmt.Fprintf(w, prefix+"\n"+prefix+"> %s [%s] @ %s:\n%s\n",
				sd.GetUserForMessage(&message), message.User,
				t.Format("02/01/2006 15:04:05 Z0700"),
				prefix+message.Text,
			)
		}
		if len(message.ThreadReplies) > 0 {
			if err := generateText(w, sd, message.ThreadReplies, "|   "); err != nil {
				return err
			}
		}
		prevMsg = message
		prevTime = t
	}
	return nil
}

// GetUserForMessage returns username for the message
func (sd *SlackDumper) GetUserForMessage(msg *Message) string {
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

// populateThreads scans the message slice for threads, if and when it
// discovers the message with ThreadTimestamp, it fetches all messages in that
// thread updating them to the msgs slice.  Returns the count of messages that
// contained threads.
//
// ref: https://api.slack.com/messaging/retrieving
func (sd *SlackDumper) populateThreads(ctx context.Context, l *rate.Limiter, msgs []Message, channelID string) (int, error) {
	total := 0
	for i := range msgs {
		if msgs[i].ThreadTimestamp == "" {
			continue
		}
		threadMsgs, err := sd.dumpThread(ctx, l, channelID, msgs[i].ThreadTimestamp)
		if err != nil {
			return total, err
		}
		msgs[i].ThreadReplies = threadMsgs
		total++
	}
	return total, nil
}

func (sd *SlackDumper) DumpThread(ctx context.Context, channelID, threadTS string) (*Conversation, error) {
	if threadTS == "" || channelID == "" {
		return nil, errors.New("internal error: channelID and threadTS are empty")
	}

	threadMsgs, err := sd.dumpThread(ctx, sd.limiter(tier3), channelID, threadTS)
	if err != nil {
		return nil, err
	}
	var filesC = make(chan *slack.File, filesCbufSz)
	dlDoneC, err := sd.newFileDownloader(ctx, sd.limiter(noTier), channelID, filesC)
	if err != nil {
		return nil, err
	}
	sd.pipeFiles(filesC, threadMsgs)
	if sd.options.dumpfiles {
		close(filesC)
		<-dlDoneC
	}

	sortMessages(threadMsgs)

	return &Conversation{
		Messages: threadMsgs,
		ID:       channelID,
		ThreadTS: threadTS,
	}, nil
}

// dumpThread retrieves all messages in the thread and returns them as a slice of
// messages.
func (sd *SlackDumper) dumpThread(ctx context.Context, l *rate.Limiter, channelID string, threadTS string) ([]Message, error) {
	var thread []Message

	var cursor string
	for {
		var (
			msgs       []slack.Message
			hasmore    bool
			nextCursor string
		)
		if err := withRetry(ctx, l, sd.options.conversationRetries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationRepliesContext", func() {
				msgs, hasmore, nextCursor, err = sd.client.GetConversationRepliesContext(
					ctx,
					&slack.GetConversationRepliesParameters{ChannelID: channelID, Timestamp: threadTS, Cursor: cursor},
				)
			})
			return err
		}); err != nil {
			return nil, err
		}

		thread = append(thread, sd.convertMsgs(msgs[1:])...) // exclude the first message of the thread, as it's the same as the parent.
		if !hasmore {
			break
		}
		cursor = nextCursor
	}
	return thread, nil
}

// convertMsgs converts a slice of slack.Message to []Message.
func (sd *SlackDumper) convertMsgs(sm []slack.Message) []Message {
	msgs := make([]Message, len(sm))
	for i := range sm {
		msgs[i].Message = sm[i]
	}
	return msgs
}
