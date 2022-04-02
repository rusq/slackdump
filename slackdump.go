package slackdump

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"time"
	"unicode/utf8"

	cookiemonster "github.com/MercuryEngineering/CookieMonster"
	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"
)

const defNumAttempts = 3 // default number of attempts for withRetry.

//go:generate mockgen -destination internal/mock_os/mock_os.go os FileInfo
//go:generate sh -c "mockgen -source slackdump.go -destination clienter_mock.go -package slackdump -mock_names clienter=mockClienter,Reporter=mockReporter"
//go:generate sed -i ~ -e "s/NewmockClienter/newmockClienter/g" -e "s/NewmockReporter/newmockReporter/g" clienter_mock.go

// SlackDumper stores basic session parameters.
type SlackDumper struct {
	client clienter

	teamID string // used as a suffix for cached users

	// Users contains the list of users and populated on NewSlackDumper
	Users     Users                  `json:"users"`
	UserIndex map[string]*slack.User `json:"-"`

	options Options
}

// clienter is the interface with some functions of slack.Client with the sole
// purpose of mocking in tests (see client_mock.go)
type clienter interface {
	GetConversationInfoContext(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetFile(downloadURL string, writer io.Writer) error
	GetTeamInfo() (*slack.TeamInfo, error)
	GetUsersContext(ctx context.Context) ([]slack.User, error)
}

// tier represents rate limit tier:
// https://api.slack.com/docs/rate-limits
type tier int

const (
	// base throttling defined as events per minute
	noTier tier = 1000 // no tier is applied

	tier1 tier = 1
	tier2 tier = 20
	tier3 tier = 50
	tier4 tier = 100
)

// AllChanTypes enumerates all API-supported channel types as of 03/2022.
var AllChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

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

func makeSlakeOpts(cookie string) ([]slack.Option, error) {
	if !isExistingFile(cookie) {
		return []slack.Option{
			slack.OptionAuthCookie(cookie),
			slack.OptionCookie("d-s", fmt.Sprintf("%d", time.Now().Unix()-10)),
		}, nil
	}
	dlog.Debug("cookie value appears to be an existing file")
	cookies, err := cookiemonster.ParseFile(cookie)
	if err != nil {
		return nil, fmt.Errorf("error loading cookies file: %w", errors.WithStack(err))
	}
	return []slack.Option{slack.OptionCookieRAW(cookies...)}, nil
}

func NewWithOptions(ctx context.Context, token string, cookie string, opts Options) (*SlackDumper, error) {
	ctx, task := trace.NewTask(ctx, "NewWithOptions")
	defer task.End()

	trace.Logf(ctx, "startup", "has_token=%v has_cookie=%v cookie_is_file=%v", token != "", cookie != "", isExistingFile(cookie))

	sopts, err := makeSlakeOpts(cookie)
	if err != nil {
		return nil, err
	}

	cl := slack.New(token, sopts...)
	ti, err := cl.GetTeamInfoContext(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sd := &SlackDumper{
		client:  cl,
		options: opts,
		teamID:  ti.ID,
	}

	dlog.Println("> checking user cache...")
	users, err := sd.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}

	sd.Users = users
	sd.UserIndex = users.IndexByID()

	return sd, nil
}

func (sd *SlackDumper) limiter(t tier) *rate.Limiter {
	return newLimiter(t, sd.options.Tier3Burst, int(sd.options.Tier3Boost))
}

func isExistingFile(cookie string) bool {
	fi, err := os.Stat(cookie)
	return err == nil && !fi.IsDir()
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
		var err error
		trace.WithRegion(ctx, "withRetry.wait", func() {
			err = l.Wait(ctx)
		})
		if err != nil {
			return err
		}

		err = fn()
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
		dlog.Print(msg)

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
