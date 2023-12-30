package slackdump

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"time"

	"errors"

	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/chttp"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
)

//go:generate mockgen -destination internal/mocks/mock_os/mock_os.go os FileInfo
//go:generate mockgen -destination internal/mocks/mock_downloader/mock_downloader.go github.com/rusq/slackdump/v2/downloader Downloader
//go:generate sh -c "mockgen -source slackdump.go -destination clienter_mock_test.go -package slackdump -mock_names clienter=mockClienter,Reporter=mockReporter"
//go:generate sed -i ~ -e "s/NewmockClienter/newmockClienter/g" -e "s/NewmockReporter/newmockReporter/g" clienter_mock_test.go

// Session stores basic session parameters.
type Session struct {
	client clienter // Slack client

	wspInfo *slack.AuthTestResponse // workspace info

	fs fsadapter.FS // filesystem for saving attachments

	// Users contains the list of users and populated on NewSession
	Users     types.Users          `json:"users"`
	UserIndex structures.UserIndex `json:"-"`

	options Options
}

// clienter is the interface with some functions of slack.Client with the sole
// purpose of mocking in tests (see client_mock.go)
type clienter interface {
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetFile(downloadURL string, writer io.Writer) error
	GetTeamInfo() (*slack.TeamInfo, error)
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
	GetEmojiContext(ctx context.Context) (map[string]string, error)
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)
}

var (
	// ErrNoUserCache is returned when the user cache is not available.
	ErrNoUserCache = errors.New("user cache unavailable")
)

// AllChanTypes enumerates all API-supported channel [types] as of 03/2023.
//
// [types]: https://api.slack.com/methods/conversations.list#arg_types
var AllChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// New creates new session with the default options  and populates the internal
// cache of users and channels for lookups.
func New(ctx context.Context, creds auth.Provider, opts ...Option) (*Session, error) {
	options := DefOptions
	for _, opt := range opts {
		opt(&options)
	}

	return NewWithOptions(ctx, creds, options)
}

// New creates new Session with provided options, and populates the internal
// cache of users and channels for lookups.  If it fails to authenticate,
// AuthError is returned.
func NewWithOptions(ctx context.Context, authProvider auth.Provider, opts Options) (*Session, error) {
	ctx, task := trace.NewTask(ctx, "NewWithOptions")
	defer task.End()

	if err := authProvider.Validate(); err != nil {
		return nil, err
	}

	httpCl, err := chttp.New("https://slack.com", authProvider.Cookies())
	if err != nil {
		return nil, err
	}

	cl := slack.New(authProvider.SlackToken(), slack.OptionHTTPClient(httpCl))

	authTestResp, err := cl.AuthTestContext(ctx)
	if err != nil {
		return nil, &auth.Error{Err: err}
	}

	sd := &Session{
		client:  cl,
		options: opts,
		wspInfo: authTestResp,
		fs:      fsadapter.NewDirectory("."), // default is to save attachments to the current directory.
	}

	network.SetLogger(sd.l())

	if err := os.MkdirAll(opts.CacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create the cache directory: %s", err)
	}

	sd.l().Println("> checking user cache...")
	users, err := sd.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}

	sd.Users = users
	sd.UserIndex = users.IndexByID()

	return sd, nil
}

// TestAuth attempts to authenticate with the given provider.  It will return
// AuthError if faled.
func TestAuth(ctx context.Context, provider auth.Provider) error {
	ctx, task := trace.NewTask(ctx, "TestAuth")
	defer task.End()

	httpCl, err := chttp.New("https://slack.com", provider.Cookies())
	if err != nil {
		return err
	}

	cl := slack.New(provider.SlackToken(), slack.OptionHTTPClient(httpCl))

	region := trace.StartRegion(ctx, "AuthTestContext")
	defer region.End()
	if _, err := cl.AuthTestContext(ctx); err != nil {
		return &auth.Error{Err: err}
	}
	return nil
}

// Client returns the underlying slack.Client.
func (sd *Session) Client() *slack.Client {
	return sd.client.(*slack.Client)
}

// Me returns the current authenticated user in a rather dirty manner.
// If the user cache is unitnitialised, it returns ErrNoUserCache.
func (sd *Session) Me() (slack.User, error) {
	if len(sd.UserIndex) == 0 {
		return slack.User{}, ErrNoUserCache
	}
	return *sd.UserIndex[sd.CurrentUserID()], nil
}

func (sd *Session) CurrentUserID() string {
	return sd.wspInfo.UserID
}

// SetFS sets the filesystem to save attachments to (slackdump defaults to the
// current directory otherwise).
func (sd *Session) SetFS(fs fsadapter.FS) {
	if fs == nil {
		return
	}
	sd.fs = fs
}

func (sd *Session) limiter(t network.Tier) *rate.Limiter {
	return network.NewLimiter(t, sd.options.Tier3Burst, int(sd.options.Tier3Boost))
}

func checkCacheFile(filename string, maxAge time.Duration) error {
	if filename == "" {
		return errors.New("no cache filename")
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}

	return validateCache(fi, maxAge)
}

func validateCache(fi os.FileInfo, maxAge time.Duration) error {
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

// l returns the current logger.
func (sd *Session) l() logger.Interface {
	if sd.options.Logger == nil {
		return logger.Default
	}
	return sd.options.Logger
}
