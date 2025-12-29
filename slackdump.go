package slackdump

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"
	"time"

	"golang.org/x/time/rate"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	st "github.com/rusq/slackdump/v3/internal/structures"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/client"
	"github.com/rusq/slackdump/v3/internal/network"
	"github.com/rusq/slackdump/v3/stream"
)

//go:generate mockgen -destination internal/mocks/mock_os/mock_os.go os FileInfo

// Session stores basic session parameters.  Zero value is not usable, must be
// initialised with New.
type Session struct {
	client client.SlackClienter // client is the Slack client to use for API calls.
	uc     *usercache           // usercache contains the list of users.
	fs     fsadapter.FS         // filesystem adapter
	log    *slog.Logger         // logger

	wspInfo *WorkspaceInfo // workspace info

	cfg config
}

// WorkspaceInfo is an type alias for [slack.AuthTestResponse].
type WorkspaceInfo = slack.AuthTestResponse

// ErrNoUserCache is returned when the user cache is not initialised.
var ErrNoUserCache = errors.New("user cache unavailable")

// AllChanTypes enumerates all API-supported channel [types] as of 12/2025.
//
// [types]: https://api.slack.com/methods/conversations.list#arg_types
var AllChanTypes = []string{st.CMPIM, st.CIM, st.CPublic, st.CPrivate}

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
func WithLimits(l network.Limits) Option {
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
func WithLogger(l *slog.Logger) Option {
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
func WithSlackClient(cl client.SlackClienter) Option {
	return func(s *Session) {
		s.client = cl
	}
}

func WithForceEnterprise(b bool) Option {
	return func(s *Session) {
		s.cfg.forceEnterprise = b
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

	return NewNoValidate(ctx, prov, opts...)
}

// NewNoValidate creates new Slackdump session with provided options, and
// populates the internal cache of users and channels for lookups.  This
// function does not validate the auth.Provider.
func NewNoValidate(ctx context.Context, prov auth.Provider, opts ...Option) (*Session, error) {
	sd := &Session{
		cfg: defConfig,
		uc:  new(usercache),

		log: slog.Default(),
	}
	for _, opt := range opts {
		opt(sd)
	}

	if err := sd.initClient(ctx, prov, sd.cfg.forceEnterprise); err != nil {
		return nil, err
	}

	return sd, nil
}

// initWorkspaceInfo gets from the API and sets the workspace information for
// the session.
func (s *Session) initWorkspaceInfo(ctx context.Context, cl client.Slack) error {
	info, err := cl.AuthTestContext(ctx)
	if err != nil {
		return err
	}
	s.wspInfo = info
	return nil
}

// initClient initialises the client that is appropriate for the current
// workspace.  It will use the initialised auth.Provider for credentials.  If
// forceEdge is true, it will use th edge client regardless of whether it
// detects the enterprise instance or not.  If the client was set by the
// WithClient option, it will not override it.
func (s *Session) initClient(ctx context.Context, prov auth.Provider, forceEdge bool) error {
	if s.client == nil {
		cl, err := client.New(ctx, prov, client.WithEnterprise(forceEdge))
		if err != nil {
			return err
		}
		s.client = cl
	}
	return s.initWorkspaceInfo(ctx, s.client)
}

// Client returns the underlying slack.Client.
func (s *Session) Client() *slack.Client {
	cl, ok := s.client.Client()
	if !ok {
		panic("client is not a slack.Client")
	}
	return cl
}

// CurrentUserID returns the user ID of the authenticated user.
func (s *Session) CurrentUserID() string {
	return s.wspInfo.UserID
}

func (s *Session) limiter(t network.Tier) *rate.Limiter {
	var tl network.TierLimit
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
	return network.NewLimiter(t, tl.Burst, int(tl.Boost))
}

// Info returns a workspace information.  Slackdump retrieves workspace
// information during the initialisation when performing authentication test,
// so no API call is involved at this point.
func (s *Session) Info() *WorkspaceInfo {
	return s.wspInfo
}

// Stream streams the channel, calling proc functions for each chunk.
func (s *Session) Stream(opts ...stream.Option) *stream.Stream {
	return stream.New(s.client, s.cfg.limits, opts...)
}
