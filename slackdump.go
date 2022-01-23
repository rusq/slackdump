package slackdump

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"runtime/trace"
	"sort"
	"time"

	"github.com/pkg/errors"

	"github.com/rusq/dlog"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

// SlackDumper stores basic session parameters.
type SlackDumper struct {
	client *slack.Client

	// Users contains the list of users and populated on NewSlackDumper
	Users     Users                  `json:"users"`
	Channels  []slack.Channel        `json:"channels"`
	UserForID map[string]*slack.User `json:"-"`

	options options
}

type options struct {
	dumpfiles           bool
	workers             int
	conversationRetries int
	downloadRetries     int
}

var allChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(w io.Writer) error
}

type Option func(*SlackDumper)

// DumpFiles controls the file download behaviour.
func DumpFiles(b bool) Option {
	return func(sd *SlackDumper) {
		sd.options.dumpfiles = b
	}
}

// RetryThreads sets the number of attempts when dumping conversations and
// threads, and getting rate limited.
func RetryThreads(attempts int) Option {
	return func(sd *SlackDumper) {
		if attempts > 0 {
			sd.options.conversationRetries = attempts
		}
	}
}

// RetryDownloads sets the number of attempts to download a file when getting
// rate limited.
func RetryDownloads(attempts int) Option {
	return func(sd *SlackDumper) {
		if attempts > 0 {
			sd.options.downloadRetries = attempts
		}
	}
}

const defNumWorkers = 4 // seems reasonable

// NumWorkers allows to set the number of file download workers. n should be in
// range [1, NumCPU]. If not in range, will be reset to a defNumWorkers number,
// which seems reasonable.
func NumWorkers(n int) Option {
	return func(sd *SlackDumper) {
		if n < 1 || runtime.NumCPU() < n {
			n = defNumWorkers
		}
		sd.options.workers = n
	}
}

// New creates new client and populates the internal cache of users and channels
// for lookups.
func New(ctx context.Context, token string, cookie string, opts ...Option) (*SlackDumper, error) {
	sd := &SlackDumper{
		client: slack.New(token, slack.OptionCookie(cookie)),
		options: options{
			workers:             defNumWorkers,
			conversationRetries: 3,
			downloadRetries:     3,
		},
	}
	for _, opt := range opts {
		opt(sd)
	}

	errC := make(chan error, 1)

	var chans *Channels

	go func() {
		defer close(errC)

		var err error
		chanTypes := allChanTypes
		dlog.Println("> caching channels, might take a while...")
		chans, err = sd.getChannels(ctx, chanTypes)
		if err != nil {
			errC <- err
		}
	}()

	dlog.Println("> caching users...")
	if _, err := sd.GetUsers(); err != nil {
		return nil, fmt.Errorf("error fetching users: %s", err)
	}

	if err := <-errC; err != nil {
		return nil, fmt.Errorf("error fetching channels: %s", err)
	}

	sd.Channels = chans.Channels

	return sd, nil
}

// IsDeletedUser checks if the user is deleted and returns appropriate value
func (sd *SlackDumper) IsDeletedUser(id string) bool {
	thisUser, ok := sd.UserForID[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}

// DumpMessages fetches messages from the conversation identified by channelID.
func (sd *SlackDumper) DumpMessages(ctx context.Context, channelID string) (*Channel, error) {
	ctx, task := trace.NewTask(ctx, "DumpMessages")
	defer task.End()

	var filesC = make(chan *slack.File, 20)

	var (
		// slack rate limits are per method, so we're safe to use different limiters for different mehtods.
		convLimiter   = newLimiter(tier3)
		threadLimiter = newLimiter(tier3)
		dlLimiter     = newLimiter(noTier) // go-slack/slack just sends the Post to the file endpoint, so this should work.
	)

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
			resp, err = sd.client.GetConversationHistoryContext(
				ctx,
				&slack.GetConversationHistoryParameters{
					ChannelID: channelID,
					Cursor:    cursor,
				},
			)
			return err
		}); err != nil {
			return nil, err
		}

		chunk := sd.convertMsgs(resp.Messages)
		if err := sd.populateThreads(ctx, threadLimiter, chunk, channelID); err != nil {
			return nil, err
		}
		sd.pipeFiles(filesC, chunk)
		messages = append(messages, chunk...)

		dlog.Printf("request #%d, fetched: %d, total: %d\n",
			i, len(resp.Messages), len(messages))

		if !resp.HasMore {
			break
		}

		cursor = resp.ResponseMetaData.NextCursor
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	if sd.options.dumpfiles {
		close(filesC)
		<-dlDoneC
	}

	return &Channel{Messages: messages, ID: channelID}, nil
}

// convertMsgs converts a slice of slack.Message to []Message.
func (sd *SlackDumper) convertMsgs(sm []slack.Message) []Message {
	msgs := make([]Message, len(sm))
	for i := range sm {
		msgs[i].Message = sm[i]
	}
	return msgs
}

// pipeFiles scans the messages and sends all the files discovered to the filesC.
func (sd *SlackDumper) pipeFiles(filesC chan<- *slack.File, msgs []Message) {
	if !sd.options.dumpfiles {
		return
	}
	// place files in download queue
	fileChunk := sd.filesFromMessages(msgs)
	for i := range fileChunk {
		filesC <- &fileChunk[i]
	}
}

// populateThreads scans the message slice for threads, if and when it
// discovers the message with ThreadTimestamp, it fetches all messages in that
// thread updating them to the msgs slice.
//
// ref: https://api.slack.com/messaging/retrieving
func (sd *SlackDumper) populateThreads(ctx context.Context, l *rate.Limiter, msgs []Message, channelID string) error {
	for i := range msgs {
		if msgs[i].ThreadTimestamp == "" {
			continue
		}
		threadMsgs, err := sd.dumpThread(ctx, l, channelID, msgs[i].ThreadTimestamp)
		if err != nil {
			return err
		}
		msgs[i].ThreadReplies = threadMsgs
	}
	return nil
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
			msgs, hasmore, nextCursor, err = sd.client.GetConversationRepliesContext(
				ctx,
				&slack.GetConversationRepliesParameters{ChannelID: channelID, Timestamp: threadTS, Cursor: cursor},
			)
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

var ErrRetryFailed = errors.New("callback was not able to complete without errors within the allowed retries count")

// withRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func withRetry(ctx context.Context, l *rate.Limiter, maxAttempts int, fn func() error) error {
	var ok bool
	for attempt := 0; attempt < maxAttempts; attempt++ {
		trace.WithRegion(ctx, "withRetry.wait", func() {
			l.Wait(ctx)
		})

		err := fn()
		if err == nil {
			ok = true
			break
		}

		trace.Logf(ctx, "error", "slackRetry: %s", err)
		var rle *slack.RateLimitedError
		if !errors.As(err, &rle) {
			return errors.WithStack(err)
		}

		trace.Logf(ctx, "info", "got rate limited, sleeping %s", rle.RetryAfter)
		time.Sleep(rle.RetryAfter)
	}
	if !ok {
		return ErrRetryFailed
	}
	return nil
}
