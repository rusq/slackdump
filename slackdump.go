package slackdump

import (
	"context"
	"fmt"
	"io"
	"runtime/trace"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// tier represents rate limit tier:
// https://api.slack.com/docs/rate-limits
type tier int

const (
	// base throttling defined as events per minute
	tier1  tier = 1
	tier2  tier = 20
	tier3  tier = 50
	tier4  tier = 100
	noTier tier = 0
)

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

var ErrRetryFailed = errors.New("callback was not able to complete without errors within the allowed number of retries")

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

// newLimiter returns throttler with rateLimit requests per minute.
// optionally caller may specify the boost
func newLimiter(t tier, burst uint, boost int) *rate.Limiter {
	callsPerSec := float64(int(t)+boost) / 60.0
	l := rate.NewLimiter(rate.Limit(callsPerSec), int(burst))
	return l
}

func fromSlackTime(timestamp string) (time.Time, error) {
	strTime := strings.Split(timestamp, ".")
	var hi, lo int64

	hi, err := strconv.ParseInt(strTime[0], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if len(strTime) > 1 {
		lo, err = strconv.ParseInt(strTime[1], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
	}
	t := time.Unix(hi, lo).UTC()
	return t, nil
}

func maxStringLength(strings []string) (maxlen int) {
	for i := range strings {
		l := utf8.RuneCountInString(strings[i])
		if l > maxlen {
			maxlen = l
		}
	}
	return
}
