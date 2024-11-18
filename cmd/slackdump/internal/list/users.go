package list

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/osext"
	"github.com/rusq/slackdump/v3/types"
)

var CmdListUsers = &base.Command{
	Run:         runListUsers,
	UsageLine:   "slackdump list users [flags] [filename]",
	PrintFlags:  true,
	FlagMask:    cfg.OmitDownloadFlag,
	Short:       "list workspace users",
	RequireAuth: true,
	Long: fmt.Sprintf(`
# List Users

List users lists workspace users in the desired format.

Users are cached for %v.  To disable caching, use '-no-user-cache' flag and
'-user-cache-retention' flag to control the caching behaviour.
`+
		sectListFormat, cfg.UserCacheRetention),
}

func init() {
	CmdListUsers.Wizard = wizUsers
}

func runListUsers(ctx context.Context, cmd *base.Command, args []string) error {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	var l = &users{
		common: commonFlags,
	}

	return list(ctx, sess, l, filename)
}

type users struct {
	data types.Users

	common commonOpts
}

func (u *users) Type() string {
	return "users"
}

func (u *users) Data() types.Users {
	return u.data
}

func (u *users) Users() []slack.User {
	return nil
}

func (u *users) Retrieve(ctx context.Context, sess *slackdump.Session, m *cache.Manager) error {
	users, err := fetchUsers(ctx, sess, m, cfg.NoUserCache, sess.Info().TeamID)
	if err != nil {
		return err
	}
	u.data = users
	return nil
}

//go:generate mockgen -source=users.go -destination=mocks_test.go -package=list userGetter,userCacher

type userGetter interface {
	GetUsers(ctx context.Context) (types.Users, error)
}

type userCacher interface {
	LoadUsers(teamID string, retention time.Duration) ([]slack.User, error)
	CacheUsers(teamID string, users []slack.User) error
}

func fetchUsers(ctx context.Context, ug userGetter, m userCacher, skipCache bool, teamID string) ([]slack.User, error) {
	lg := cfg.Log.With("team_id", teamID, "cache", !skipCache)

	if !skipCache {
		// attempt to load from cache
		users, err := m.LoadUsers(teamID, cfg.UserCacheRetention)
		if err == nil {
			return users, nil
		}

		// failed to load from cache
		if !errors.Is(err, cache.ErrExpired) && !errors.Is(err, cache.ErrEmpty) && !os.IsNotExist(err) && !osext.IsPathError(err) {
			// some funky error
			return nil, err
		}
		lg.InfoContext(ctx, "user cache expired or empty, caching users")
	}
	// getting users from API
	users, err := ug.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	// saving users to cache, will ignore any errors, but notify the user.
	if err := m.CacheUsers(teamID, users); err != nil {
		lg.WarnContext(ctx, "failed saving user cache (ignored)", "error", err)
	}

	return users, nil
}
