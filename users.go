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

// In this file: user related code.

import (
	"context"
	"errors"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/types"
)

const (
	cacheTimeout = 5 * time.Minute
)

// GetUsers retrieves all users either from cache or from the API.  If
// Session.usercache is not empty, it will return the cached users.
// Otherwise, it will try fetching them from the API and cache them.
func (s *Session) GetUsers(ctx context.Context) (types.Users, error) {
	ctx, task := trace.NewTask(ctx, "GetUsers")
	defer task.End()

	// try getting users from cache
	users, err := s.uc.get(cacheTimeout)
	if err == nil {
		return users, nil
	}

	// if not succeeded, fetch them from the API.
	users, err = s.fetchUsers(ctx)
	if err != nil {
		return nil, err
	}
	s.uc.set(users)
	return users, err
}

// fetchUsers fetches users from the API.
func (s *Session) fetchUsers(ctx context.Context) (types.Users, error) {
	var users []slack.User

	l := s.limiter(network.Tier2)
	if err := network.WithRetry(ctx, l, s.cfg.limits.Tier2.Retries, func(ctx context.Context) error {
		var err error
		users, err = s.client.GetUsersContext(ctx)
		return err
	}); err != nil {
		trace.Logf(ctx, "error", "fetchUsers error=%s", err)
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("couldn't fetch users")
	}
	return users, nil
}
