package slackdump

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/trace"
	"time"

	"github.com/pkg/errors"
	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

//go:generate mockgen -destination internal/mocks/mock_os/mock_os.go os FileInfo
//go:generate mockgen -destination internal/mocks/mock_downloader/mock_downloader.go github.com/rusq/slackdump/v2/downloader Downloader
//go:generate sh -c "mockgen -source slackdump.go -destination clienter_mock_test.go -package slackdump -mock_names clienter=mockClienter,Reporter=mockReporter"
//go:generate sed -i ~ -e "s/NewmockClienter/newmockClienter/g" -e "s/NewmockReporter/newmockReporter/g" clienter_mock_test.go

const (
	// user index of the application user in the user list.
	userIdxMe = 1

	cacheDirName = "slackdump"
)

// SlackDumper stores basic session parameters.
type SlackDumper struct {
	client clienter

	teamID string // used as a suffix for cached users

	fs fsadapter.FS // filesystem for saving attachments

	// Users contains the list of users and populated on NewSlackDumper
	Users     types.Users          `json:"users"`
	UserIndex structures.UserIndex `json:"-"`
	me        slack.User

	options Options

	cacheDir string // cache directory on local system
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

// AllChanTypes enumerates all API-supported channel types as of 03/2022.
var AllChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Reporter is an interface defining output functions
type Reporter interface {
	ToText(w io.Writer, ui structures.UserIndex) error
}

// New creates new client and populates the internal cache of users and channels
// for lookups.
func New(ctx context.Context, creds auth.Provider, opts ...Option) (*SlackDumper, error) {
	options := DefOptions
	for _, opt := range opts {
		opt(&options)
	}

	return NewWithOptions(ctx, creds, options)
}

func (sd *SlackDumper) Client() *slack.Client {
	return sd.client.(*slack.Client)
}

func NewWithOptions(ctx context.Context, authProvider auth.Provider, opts Options) (*SlackDumper, error) {
	ctx, task := trace.NewTask(ctx, "NewWithOptions")
	defer task.End()

	if err := authProvider.Validate(); err != nil {
		return nil, err
	}

	cl := slack.New(authProvider.SlackToken(), slack.OptionCookieRAW(toPtrCookies(authProvider.Cookies())...))
	ti, err := cl.GetTeamInfoContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting team information: %w", err)
	}

	cacheDir, err := createCacheDir(cacheDirName)
	if err != nil {
		cacheDir = "."
		dlog.Printf("failed to create the cache directory, will use current")
	}

	sd := &SlackDumper{
		client:   cl,
		options:  opts,
		teamID:   ti.ID,
		fs:       fsadapter.NewDirectory("."), // default is to save attachments to the current directory.
		cacheDir: cacheDir,
	}

	dlog.Println("> checking user cache...")
	users, err := sd.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching users: %w", err)
	}
	if len(users) < 2 { // me and slackbot.
		return nil, errors.New("invalid number of users retrieved.")
	}

	// now, this is filthy, but Slack does not allow us to call GetUserIdentity with browser token.
	sd.me = users[userIdxMe]

	sd.Users = users
	sd.UserIndex = users.IndexByID()

	return sd, nil
}

func createCacheDir(subdir string) (string, error) {
	if subdir == "" {
		return "", errors.New("can't use top level cache directory")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	cachePath := filepath.Join(cache, subdir)
	if err := os.MkdirAll(cachePath, 0750); err != nil {
		return "", err
	}
	return cachePath, nil
}

func (sd *SlackDumper) Me() slack.User {
	return sd.me
}

// SetFS sets the filesystem to save attachments to (slackdump defaults to the
// current directory otherwise).
func (sd *SlackDumper) SetFS(fs fsadapter.FS) {
	if fs == nil {
		return
	}
	sd.fs = fs
}

func toPtrCookies(cc []http.Cookie) []*http.Cookie {
	var ret = make([]*http.Cookie, len(cc))
	for i := range cc {
		ret[i] = &cc[i]
	}
	return ret
}

func (sd *SlackDumper) limiter(t network.Tier) *rate.Limiter {
	return network.NewLimiter(t, sd.options.Tier3Burst, int(sd.options.Tier3Boost))
}

// withRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func withRetry(ctx context.Context, l *rate.Limiter, maxAttempts int, fn func() error) error {
	return network.WithRetry(ctx, l, maxAttempts, fn)
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
