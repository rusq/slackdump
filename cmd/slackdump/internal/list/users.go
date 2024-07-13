package list

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/internal/osext"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/types"
)

var CmdListUsers = &base.Command{
	Run:        listUsers,
	Wizard:     wizUsers,
	UsageLine:  "slackdump list users [flags] [filename]",
	PrintFlags: true,
	FlagMask:   cfg.OmitDownloadFlag,
	Short:      "list workspace users",
	Long: fmt.Sprintf(`
# List Users

List users lists workspace users in the desired format.

Users are cached for %v.  To disable caching, use '-no-user-cache' flag and
'-user-cache-retention' flag to control the caching behaviour.
`+
		sectListFormat, cfg.UserCacheRetention),
	RequireAuth: true,
}

func listUsers(ctx context.Context, cmd *base.Command, args []string) error {
	if err := list(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, ".json")
		if len(args) > 0 {
			filename = args[0]
		}
		users, err := sess.GetUsers(ctx)
		return users, filename, err
	}); err != nil {
		return err
	}
	return nil
}

func wizUsers(ctx context.Context, cmd *base.Command, args []string) error {
	return wizard(ctx, func(ctx context.Context, sess *slackdump.Session) (any, string, error) {
		var filename = makeFilename("users", sess.Info().TeamID, ".json")
		users, err := sess.GetUsers(ctx)
		return users, filename, err
	})
}

//go:generate mockgen -source=users.go -destination=mocks_test.go -package=list userGetter,userCacher

type userGetter interface {
	GetUsers(ctx context.Context) (types.Users, error)
}

type userCacher interface {
	LoadUsers(teamID string, retention time.Duration) ([]slack.User, error)
	CacheUsers(teamID string, users []slack.User) error
}

func getCachedUsers(ctx context.Context, ug userGetter, m userCacher, teamID string) ([]slack.User, error) {
	lg := logger.FromContext(ctx)

	users, err := m.LoadUsers(teamID, cfg.UserCacheRetention)
	if err == nil {
		return users, nil
	}

	// failed to load from cache
	if !errors.Is(err, cache.ErrExpired) && !errors.Is(err, cache.ErrEmpty) && !os.IsNotExist(err) && !osext.IsPathError(err) {
		// some funky error
		return nil, err
	}
	lg.Println("user cache expired or empty, caching users")

	// getting users from API
	users, err = ug.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	// saving users to cache, will ignore any errors, but notify the user.
	if err := m.CacheUsers(teamID, users); err != nil {
		trace.Logf(ctx, "error", "saving user cache to %q, error: %s", userCacheBase, err)
		lg.Printf("warning: failed saving user cache to %q: %s, but nevermind, let's continue", userCacheBase, err)
	}

	return users, nil
}
