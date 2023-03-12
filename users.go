package slackdump

// In this file: user related code.

import (
	"context"
	"errors"
	"runtime/trace"
	"sync"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

type usercache struct {
	users    types.Users
	mu       sync.RWMutex
	cachedAt time.Time
}

var errCacheExpired = errors.New("cache expired")

// get retrieves users from cache.  If cache is empty or expired, it will
// return errCacheExpired.
func (uc *usercache) get(retention time.Duration) (types.Users, error) {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	if len(uc.users) > 0 && time.Since(uc.cachedAt) < retention {
		return uc.users, nil
	}
	return nil, errCacheExpired
}

func (uc *usercache) set(users types.Users) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.users = users
	uc.cachedAt = time.Now()
}

// GetUsers retrieves all users either from cache or from the API.  If
// Session.usercache is not empty, it will return the cached users.
// Otherwise, it will try fetching them from the API and cache them.
func (s *Session) GetUsers(ctx context.Context) (types.Users, error) {
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	users, err := s.uc.get(s.cfg.UserCache.Retention)
	if err == nil {
		return users, nil
	}

	users, err = s.fetchUsers(ctx)
	if err != nil {
		return nil, err
	}
	s.uc.set(users)
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
