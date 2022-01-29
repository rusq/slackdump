package slackdump

import (
	"context"
	"fmt"
	"io"
	"runtime/trace"
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
	UserIndex map[string]*slack.User `json:"-"`

	options options
}

var allChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(sd *SlackDumper, w io.Writer) error
}

// New creates new client and populates the internal cache of users and channels
// for lookups.
func New(ctx context.Context, token string, cookie string, opts ...Option) (*SlackDumper, error) {
	sd := &SlackDumper{
		client:  slack.New(token, slack.OptionCookie(cookie)),
		options: defOptions,
	}
	for _, opt := range opts {
		opt(sd)
	}

	dlog.Println("> caching users...")
	if _, err := sd.GetUsers(ctx); err != nil {
		return nil, fmt.Errorf("error fetching users: %s", err)
	}

	return sd, nil
}

// IsDeletedUser checks if the user is deleted and returns appropriate value
func (sd *SlackDumper) IsDeletedUser(id string) bool {
	thisUser, ok := sd.UserIndex[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}

func (sd *SlackDumper) limiter(t tier) *rate.Limiter {
	return newLimiter(t, sd.options.limiterBurst, int(sd.options.limiterBoost))
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

		msg := fmt.Sprintf("got rate limited, sleeping %s", rle.RetryAfter)
		trace.Log(ctx, "info", msg)
		dlog.Debug(msg)

		time.Sleep(rle.RetryAfter)
	}
	if !ok {
		return ErrRetryFailed
	}
	return nil
}
