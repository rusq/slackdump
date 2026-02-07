// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
