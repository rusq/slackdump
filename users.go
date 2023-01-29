package slackdump

// In this file: user related code.

import (
	"context"
	"os"
	"runtime/trace"

	"errors"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

// GetUsers retrieves all users either from cache or from the API.
func (sd *Session) GetUsers(ctx context.Context) (types.Users, error) {
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	if sd.options.UserCache.Disabled {
		return types.Users{}, nil
	}

	// TODO make Manager a Session variable.  Maybe?
	m, err := cache.NewManager(sd.options.CacheDir, cache.WithUserBasename(sd.options.UserCache.Filename))
	if err != nil {
		return nil, err
	}

	users, err := m.LoadUsers(sd.wspInfo.TeamID, sd.options.UserCache.MaxAge)
	if err != nil {
		if os.IsNotExist(err) {
			sd.l().Println("  caching users for the first time")
		} else {
			sd.l().Printf("  %s: it will be recreated.", err)
		}
		users, err = sd.fetchUsers(ctx)
		if err != nil {
			return nil, err
		}
		if err := m.SaveUsers(sd.wspInfo.TeamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", sd.options.UserCache.Filename, err)
			sd.l().Printf("error saving user cache to %q: %s, but nevermind, let's continue", sd.options.UserCache.Filename, err)
		}
	}

	return users, err
}

// fetchUsers fetches users from the API.
func (sd *Session) fetchUsers(ctx context.Context) (types.Users, error) {
	var (
		users []slack.User
	)
	l := network.NewLimiter(
		network.Tier2, sd.options.Limits.Tier2.Burst, int(sd.options.Limits.Tier2.Boost),
	)
	if err := withRetry(ctx, l, sd.options.Limits.Tier2.Retries, func() error {
		var err error
		users, err = sd.client.GetUsersContext(ctx)
		return err
	}); err != nil {
		trace.Logf(ctx, "error", "GetUsers error=%s", err)
		return nil, err
	}
	// BUG: as of 201902 there's a bug in slack module, the invalid_auth error
	// is not propagated properly, so we'll check for number of users.  There
	// should be at least one (slackbot).
	if len(users) == 0 {
		return nil, errors.New("couldn't fetch users")
	}
	return users, nil
}
