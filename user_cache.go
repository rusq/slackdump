package slackdump

import (
	"errors"
	"sync"
	"time"

	"github.com/rusq/slackdump/v3/types"
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
