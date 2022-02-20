package slackdump

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const defNumAttempts = 3 // default number of attempts for withRetry.

//go:generate mockgen -destination internal/mock_os/mock_os.go os FileInfo
//go:generate sh -c "mockgen -source slackdump.go -destination clienter_mock.go -package slackdump -mock_names clienter=mockClienter"
//go:generate sed -i ~ "s/NewmockClienter/newmockClienter/g" clienter_mock.go

// SlackDumper stores basic session parameters.
type SlackDumper struct {
	client clienter

	// Users contains the list of users and populated on NewSlackDumper
	Users     Users                  `json:"users"`
	UserIndex map[string]*slack.User `json:"-"`

	options Options
}

// clienter is the interface with some functions of slack.Client with the sole
// purpose of mocking in tests (see client_mock.go)
type clienter interface {
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversations(params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetFile(downloadURL string, writer io.Writer) error
	GetUsers() ([]slack.User, error)
}

// tier represents rate limit tier:
// https://api.slack.com/docs/rate-limits
type tier int

const (
	// base throttling defined as events per minute
	noTier tier = 0 // no tier is applied

	tier1 tier = 1
	tier2 tier = 20
	tier3 tier = 50
	tier4 tier = 100
)

var allChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(sd *SlackDumper, w io.Writer) error
}

// New creates new client and populates the internal cache of users and channels
// for lookups.
func New(ctx context.Context, token string, cookie string, opts ...Option) (*SlackDumper, error) {
	options := DefOptions
	for _, opt := range opts {
		opt(&options)
	}

	return NewWithOptions(ctx, token, cookie, options)
}

func NewWithOptions(ctx context.Context, token string, cookie string, opts Options) (*SlackDumper, error) {
	sd := &SlackDumper{
		client:  slack.New(token, slack.OptionCookie(cookie)),
		options: opts,
	}

	dlog.Println("> checking user cache...")
	users, err := sd.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %s", err)
	}

	sd.Users = users
	sd.UserIndex = users.IndexByID()

	return sd, nil
}

func (sd *SlackDumper) limiter(t tier) *rate.Limiter {
	return newLimiter(t, sd.options.Tier3Burst, int(sd.options.Tier3Boost))
}

var ErrRetryFailed = errors.New("callback was not able to complete without errors within the allowed number of retries")

// withRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func withRetry(ctx context.Context, l *rate.Limiter, maxAttempts int, fn func() error) error {
	var ok bool
	if maxAttempts == 0 {
		maxAttempts = defNumAttempts
	}
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

func maxStringLength(strings []string) (maxlen int) {
	for i := range strings {
		l := utf8.RuneCountInString(strings[i])
		if l > maxlen {
			maxlen = l
		}
	}
	return
}

func checkCacheFile(filename string, maxAge time.Duration) error {
	if filename == "" {
		return errors.New("no cache filename")
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return validateFileStats(fi, maxAge)
}

func validateFileStats(fi os.FileInfo, maxAge time.Duration) error {
	if fi.IsDir() {
		return errors.New("cache file is a directory")
	}
	if fi.Size() == 0 {
		return errors.New("empty cache file")
	}
	if time.Since(fi.ModTime()) > maxAge {
		return errors.New("cache expired")
	}
	return nil
}
