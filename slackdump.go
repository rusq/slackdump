package slackdump

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/trace"

	"github.com/go-playground/validator/v10"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/chttp"
	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/logger"
	"github.com/rusq/slackdump/v2/types"
)

//go:generate mockgen -destination internal/mocks/mock_os/mock_os.go os FileInfo
//go:generate mockgen -destination internal/mocks/mock_downloader/mock_downloader.go github.com/rusq/slackdump/v2/downloader Downloader
//go:generate sh -c "mockgen -source slackdump.go -destination clienter_mock_test.go -package slackdump -mock_names clienter=mockClienter,Reporter=mockReporter"
//go:generate sed -i ~ -e "s/NewmockClienter/newmockClienter/g" -e "s/NewmockReporter/newmockReporter/g" clienter_mock_test.go

// Session stores basic session parameters.  Zero value is not usable, must be
// initialised with New.
type Session struct {
	client clienter // Slack client

	wspInfo *WorkspaceInfo // workspace info

	// Users contains the list of users and populated on NewSession
	Users types.Users `json:"users"`

	fs  fsadapter.FS     // filesystem adapter
	log logger.Interface // logger

	atClose []func() error // functions to call on exit

	cfg Config
}

// WorkspaceInfo is an type alias for [slack.AuthTestResponse].
type WorkspaceInfo = slack.AuthTestResponse

// clienter is the interface with some functions of slack.Client with the sole
// purpose of mocking in tests (see client_mock.go)
type clienter interface {
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetFile(downloadURL string, writer io.Writer) error
	GetTeamInfoContext(ctx context.Context) (*slack.TeamInfo, error)
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
	GetEmojiContext(ctx context.Context) (map[string]string, error)
}

var (
	// ErrNoUserCache is returned when the user cache is not initialised.
	ErrNoUserCache = errors.New("user cache unavailable")
)

// AllChanTypes enumerates all API-supported channel types as of 03/2022.
var AllChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Option is the signature of the option-setting function.
type Option func(*Session)

// WithFilesystem sets the filesystem adapter to use for the session.  If this
// option is not given, the default filesystem adapter is initialised with the
// base location specified in the Config.
func WithFilesystem(fs fsadapter.FSCloser) Option {
	return func(s *Session) {
		if fs != nil {
			s.fs = fs
		}
	}
}

// WithLogger sets the logger to use for the session.  If this option is not
// given, the default logger is initialised with the filename specified in
// Config.Logfile.  If the Config.Logfile is empty, the default logger writes
// to STDERR.
func WithLogger(l logger.Interface) Option {
	return func(s *Session) {
		if l != nil {
			s.log = l
		}
	}
}

// New creates new Slackdump session with provided options, and populates the
// internal cache of users and channels for lookups. If it fails to
// authenticate, AuthError is returned.
func New(ctx context.Context, prov auth.Provider, cfg Config, opts ...Option) (*Session, error) {
	ctx, task := trace.NewTask(ctx, "New")
	defer task.End()

	if err := cfg.Limits.Validate(); err != nil {
		var vErr validator.ValidationErrors
		if errors.As(err, &vErr) {
			return nil, fmt.Errorf("API limits failed validation: %s", vErr.Translate(OptErrTranslations))
		}
		return nil, err
	}
	if err := prov.Validate(); err != nil {
		return nil, fmt.Errorf("auth provider validation error: %s", err)
	}

	httpCl := chttp.New("https://slack.com", prov.Cookies())

	cl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(httpCl))

	authTestResp, err := cl.AuthTestContext(ctx)
	if err != nil {
		return nil, &auth.Error{Err: err}
	}

	sd := &Session{
		client:  cl,
		cfg:     cfg,
		wspInfo: authTestResp,
		log:     logger.Default,
	}
	for _, opt := range opts {
		opt(sd)
	}
	if sd.fs == nil {
		if err := sd.openFS(cfg.BaseLocation); err != nil {
			return nil, fmt.Errorf("failed to initialise filesystem adapter: %s", err)
		}
	}
	if sd.log == nil {
		if err := sd.openLogger(cfg.Logfile); err != nil {
			return nil, fmt.Errorf("failed to initialise logger: %s", err)
		}
	}

	sd.propagateLogger()

	if err := os.MkdirAll(cfg.CacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create the cache directory: %s", err)
	}

	if !sd.cfg.UserCache.Disabled {
		users, err := sd.GetUsers(ctx)
		if err != nil {
			return nil, fmt.Errorf("error fetching users: %w", err)
		}

		sd.Users = users
	}

	return sd, nil
}

// Client returns the underlying slack.Client.
func (s *Session) Client() *slack.Client {
	return s.client.(*slack.Client)
}

func (s *Session) openFS(loc string) error {
	// if no filesystem adapter is provided through Options, initialise
	// the default one.
	fs, err := fsadapter.New(loc)
	if err != nil {
		return err
	}
	s.fs = fs
	s.atClose = append(s.atClose, func() error {
		return fs.Close()
	})
	return nil
}

// Filesystem returns the filesystem adapter used by the session.
func (s *Session) Filesystem() fsadapter.FS {
	return s.fs
}

func (s *Session) openLogger(filename string) error {
	if filename == "" {
		s.log = logger.Default
		return nil
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	s.atClose = append(s.atClose, func() error {
		return f.Close()
	})
	s.log = dlog.New(f, "", log.LstdFlags, false)
	return nil
}

// Close closes the handles if they were created by the session.
// It must be called when the session is no longer needed.
func (s *Session) Close() error {
	var last error
	for _, fn := range s.atClose {
		if err := fn(); err != nil {
			last = err
		}
	}
	return last
}

// Me returns the current authenticated user in a rather dirty manner.
// If the user cache is unitnitialised, it returns ErrNoUserCache.
func (s *Session) Me() (slack.User, error) {
	if len(s.Users) == 0 {
		return slack.User{}, ErrNoUserCache
	}
	return *s.Users.IndexByID()[s.CurrentUserID()], nil
}

func (s *Session) CurrentUserID() string {
	return s.wspInfo.UserID
}

func (s *Session) limiter(t network.Tier) *rate.Limiter {
	return network.NewLimiter(t, s.cfg.Limits.Tier3.Burst, int(s.cfg.Limits.Tier3.Boost))
}

// withRetry will run the callback function fn. If the function returns
// slack.RateLimitedError, it will delay, and then call it again up to
// maxAttempts times. It will return an error if it runs out of attempts.
func withRetry(ctx context.Context, l *rate.Limiter, maxAttempts int, fn func() error) error {
	return network.WithRetry(ctx, l, maxAttempts, fn)
}

// propagateLogger propagates the slackdump logger to some dumb packages.
func (s *Session) propagateLogger() {
	network.Logger = s.log
}

// Info returns a workspace information.  Slackdump retrieves workspace
// information during the initialisation when performing authentication test,
// so no API call is involved at this point.
func (s *Session) Info() *WorkspaceInfo {
	return s.wspInfo
}
