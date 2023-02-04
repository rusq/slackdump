package slackdump

// In this file: user related code.

import (
	"context"
	"runtime/trace"

	"errors"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/cache"
	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

// GetUsers retrieves all users either from cache or from the API.
func (s *Session) GetUsers(ctx context.Context) (types.Users, error) {
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	if s.cfg.UserCache.Disabled {
		return types.Users{}, nil
	}

	// TODO make Manager a Session variable.  Maybe?
	m, err := cache.NewManager(s.cfg.CacheDir, cache.WithUserCacheBase(s.cfg.UserCache.Filename))
	if err != nil {
		return nil, err
	}

	users, err := m.LoadUsers(s.wspInfo.TeamID, s.cfg.UserCache.Retention)
	if err != nil {
		s.log.Println("caching users")
		users, err = s.fetchUsers(ctx)
		if err != nil {
			return nil, err
		}
		if err := m.SaveUsers(s.wspInfo.TeamID, users); err != nil {
			trace.Logf(ctx, "error", "saving user cache to %q, error: %s", s.cfg.UserCache.Filename, err)
			s.log.Printf("error saving user cache to %q: %s, but nevermind, let's continue", s.cfg.UserCache.Filename, err)
		}
	}

	return users, err
}

// fetchUsers fetches users from the API.
func (s *Session) fetchUsers(ctx context.Context) (types.Users, error) {
	var (
		users []slack.User
	)
	l := network.NewLimiter(
		network.Tier2, s.cfg.Limits.Tier2.Burst, int(s.cfg.Limits.Tier2.Boost),
	)
	if err := withRetry(ctx, l, s.cfg.Limits.Tier2.Retries, func() error {
		var err error
		users, err = s.client.GetUsersContext(ctx)
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
