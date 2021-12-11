package slackdump

import (
	"context"
	"fmt"
	"io"
	"log"
	"sort"

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
	dumpfiles bool
}

var allChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(w io.Writer) error
}

type Option func(*SlackDumper)

func DumpFiles(b bool) Option {
	return func(sd *SlackDumper) {
		sd.options.dumpfiles = b
	}
}

// New creates new client and populates the internal cache of users and channels
// for lookups.
func New(ctx context.Context, token string, cookie string, opts ...Option) (*SlackDumper, error) {
	var err error
	sd := &SlackDumper{
		client: slack.New(token, slack.OptionCookie(cookie)),
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
		log.Println("> caching channels, might take a while...")
		chans, err = sd.getChannels(ctx, chanTypes)
		if err != nil {
			errC <- err
		}
	}()

	log.Println("> caching users...")
	if _, err := sd.GetUsers(); err != nil {
		return nil, fmt.Errorf("error fetching users: %s", err)
	}

	if err = <-errC; err != nil {
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

	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
	}

	filesC := make(chan *slack.File, 20)
	done := make(chan bool)
	errC := make(chan error, 1)

	if sd.options.dumpfiles {
		go func() {
			errC <- sd.fileDownloader(channelID, filesC, done)
		}()
	}

	limiter := newLimiter(tier3)
	var messages []Message

	for i := 1; ; i++ {
		select {
		case err := <-errC:
			// stop the goroutine gracefully if it's running
			close(filesC)
			<-done
			return nil, err
		default:
		}
		resp, err := sd.client.GetConversationHistoryContext(ctx, params)
		if err != nil {
			return nil, err
		}
		chunk := sd.convertMsgs(resp.Messages)

		if err := sd.populateThreads(ctx, chunk, channelID, limiter); err != nil {
			return nil, err
		}
		sd.extractFiles(filesC, chunk)
		messages = append(messages, chunk...)

		log.Printf("request #%d, fetched: %d, total: %d\n",
			i, len(resp.Messages), len(messages))

		if !resp.HasMore {
			break
		}

		params.Cursor = resp.ResponseMetaData.NextCursor

		limiter.Wait(ctx)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	if sd.options.dumpfiles {
		close(filesC)
		<-done
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

// extractFiles scans the messages and sends all the files discovered to the filesC.
func (sd *SlackDumper) extractFiles(filesC chan<- *slack.File, msgs []Message) {
	if !sd.options.dumpfiles {
		return
	}
	// place files in download queue
	chunk := sd.filesFromMessages(msgs)
	for i := range chunk {
		filesC <- &chunk[i]
	}
}

// populateThreads scans the message slice for threads, if and when it
// discovers the message with ThreadTimestamp, it fetches all messages in that
// thread updating them to the msgs slice.
// ref: https://api.slack.com/messaging/retrieving
func (sd *SlackDumper) populateThreads(ctx context.Context, msgs []Message, channelID string, l *rate.Limiter) error {
	for i := range msgs {
		if msgs[i].ThreadTimestamp == "" {
			continue
		}
		threadMsgs, err := sd.dumpThread(ctx, channelID, msgs[i].ThreadTimestamp, l)
		if err != nil {
			return err
		}
		msgs[i].ThreadReplies = threadMsgs
	}
	return nil
}

// dumpThread retrieves all messages in the thread and returns them as a slice
// of messages.
func (sd *SlackDumper) dumpThread(ctx context.Context, channelID string, threadTS string, l *rate.Limiter) ([]Message, error) {
	var thread []Message

	var cursor string
	for {
		msgs, hasmore, nextCursor, err := sd.client.GetConversationRepliesContext(
			ctx,
			&slack.GetConversationRepliesParameters{ChannelID: channelID, Timestamp: threadTS, Cursor: cursor},
		)
		if err != nil {
			return nil, err
		}
		thread = append(thread, sd.convertMsgs(msgs[1:])...) // exclude the first message of the thread, as it's the same as the parent.
		if !hasmore {
			break
		}
		cursor = nextCursor
		l.Wait(ctx)
	}
	return thread, nil
}
