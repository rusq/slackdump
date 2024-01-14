package slackdump

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/trace"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/slack-go/slack"
	"golang.org/x/time/rate"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/logger"
)

//go:generate mockgen -destination internal/mocks/mock_os/mock_os.go os FileInfo
//go:generate mockgen -source slackdump.go -destination clienter_mock_test.go -package slackdump -mock_names clienter=mockClienter,Reporter=mockReporter

// Session stores basic session parameters.  Zero value is not usable, must be
// initialised with New.
type Session struct {
	client clienter         // Slack client
	uc     *usercache       // usercache contains the list of users.
	fs     fsadapter.FS     // filesystem adapter
	log    logger.Interface // logger

	wspInfo *WorkspaceInfo // workspace info

	cfg config
}

// WorkspaceInfo is an type alias for [slack.AuthTestResponse].
type WorkspaceInfo = slack.AuthTestResponse

// Slacker is the interface with some functions of slack.Client.
type Slacker interface {
	AuthTestContext(context.Context) (response *slack.AuthTestResponse, err error)
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error)
	GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)
	ListBookmarks(channelID string) ([]slack.Bookmark, error)
}

// clienter is the interface with some functions of slack.Client with the sole
// purpose of mocking in tests (see client_mock.go)
type clienter interface {
	Slacker
	GetFile(downloadURL string, writer io.Writer) error
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
	GetEmojiContext(ctx context.Context) (map[string]string, error)
}

// ErrNoUserCache is returned when the user cache is not initialised.
var ErrNoUserCache = errors.New("user cache unavailable")

// AllChanTypes enumerates all API-supported channel [types] as of 03/2023.
//
// [types]: https://api.slack.com/methods/conversations.list#arg_types
var AllChanTypes = []string{"mpim", "im", "public_channel", "private_channel"}

// Option is the signature of the option-setting function.
type Option func(*Session)

// WithFilesystem sets the filesystem adapter to use for the session.  If this
// option is not given, the default filesystem adapter is initialised with the
// base location specified in the Config.
func WithFilesystem(fs fsadapter.FS) Option {
	return func(s *Session) {
		if fs != nil {
			s.fs = fs
		}
	}
}

// WithLimits sets the API limits to use for the session.  If this option is
// not given, the default limits are initialised with the values specified in
// DefLimits.
func WithLimits(l Limits) Option {
	return func(s *Session) {
		if l.Validate() == nil {
			s.cfg.limits = l
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

// WithUserCacheRetention sets the retention period for the user cache.  If this
// option is not given, the default value is 60 minutes.
func WithUserCacheRetention(d time.Duration) Option {
	return func(s *Session) {
		s.cfg.cacheRetention = d
	}
}

// WithSlackClient sets the Slack client to use for the session.  If this
func WithSlackClient(cl clienter) Option {
	return func(s *Session) {
		s.client = cl
	}
}

// New creates new Slackdump session with provided options, and populates the
// internal cache of users and channels for lookups. If it fails to
// authenticate, AuthError is returned.
func New(ctx context.Context, prov auth.Provider, opts ...Option) (*Session, error) {
	ctx, task := trace.NewTask(ctx, "New")
	defer task.End()

	if err := prov.Validate(); err != nil {
		return nil, fmt.Errorf("auth provider validation error: %s", err)
	}

	httpCl, err := prov.HTTPClient()
	if err != nil {
		return nil, err
	}
	cl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(httpCl))

	sd := &Session{
		client: cl,
		cfg:    defConfig,
		uc:     new(usercache),

		log: logger.Default,
	}
	for _, opt := range opts {
		opt(sd)
	}
	network.SetLogger(sd.log) // set the logger for the network package

	if err := sd.cfg.limits.Validate(); err != nil {
		var vErr validator.ValidationErrors
		if errors.As(err, &vErr) {
			return nil, fmt.Errorf("API limits failed validation: %s", vErr.Translate(OptErrTranslations))
		}
		return nil, err
	}
	authResp, err := sd.client.AuthTestContext(ctx)
	if err != nil {
		return nil, &auth.Error{Err: err}
	}
	sd.wspInfo = authResp

	return sd, nil
}

// Client returns the underlying slack.Client.
func (s *Session) Client() *slack.Client {
	return s.client.(*slack.Client)
}

// CurrentUserID returns the user ID of the authenticated user.
func (s *Session) CurrentUserID() string {
	return s.wspInfo.UserID
}

func (s *Session) limiter(t network.Tier) *rate.Limiter {
	var tl TierLimit
	switch t {
	case network.Tier2:
		tl = s.cfg.limits.Tier2
	case network.Tier3:
		tl = s.cfg.limits.Tier3
	case network.Tier4:
		tl = s.cfg.limits.Tier4
	default:
		tl = s.cfg.limits.Tier3
	}
	return network.NewLimiter(t, tl.Burst, int(tl.Boost)) // BUG: tier was always 3, should fix in master too.
}

// Info returns a workspace information.  Slackdump retrieves workspace
// information during the initialisation when performing authentication test,
// so no API call is involved at this point.
func (s *Session) Info() *WorkspaceInfo {
	return s.wspInfo
}

// Stream streams the channel, calling proc functions for each chunk.
func (s *Session) Stream(opts ...StreamOption) *Stream {
	return NewStream(s.client, &s.cfg.limits, opts...)
}
