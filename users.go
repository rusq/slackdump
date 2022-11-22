package slackdump

// In this file: user related code.

import (
	"context"
	"os"
	"runtime/trace"

	"errors"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

// GetUsers retrieves all users either from cache or from the API.
func (sd *Session) GetUsers(ctx context.Context) (types.Users, error) {
	// TODO: validate that the cache is from the same workspace, it can be done by team ID.
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	if sd.options.NoUserCache {
		return types.Users{}, nil
	}

	users, err := LoadUserCache(sd.options.CacheDir, sd.options.UserCacheFilename, sd.wspInfo.TeamID, sd.options.MaxUserCacheAge)
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
		if err := SaveUserCache(sd.options.CacheDir, sd.options.UserCacheFilename, sd.wspInfo.TeamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", sd.options.UserCacheFilename, err)
			sd.l().Printf("error saving user cache to %q: %s, but nevermind, let's continue", sd.options.UserCacheFilename, err)
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
