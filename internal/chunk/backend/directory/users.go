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
package directory

import (
	"context"
	"fmt"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/processor"
)

// Users is a users processor, writes users into the users.json.gz file.
type Users struct {
	*dirproc
	cb func([]slack.User) error
}

var _ processor.Users = &Users{}

type UserOption func(*Users)

// WithUsers sets the users callback.
func WithUsers(cb func([]slack.User) error) UserOption {
	return func(u *Users) {
		u.cb = cb
	}
}

// NewUsers creates a new Users processor.
func NewUsers(cd *chunk.Directory, opt ...UserOption) (*Users, error) {
	p, err := newDirProc(cd, chunk.FUsers)
	if err != nil {
		return nil, err
	}
	u := &Users{dirproc: p}
	for _, o := range opt {
		o(u)
	}
	return u, nil
}

// Users processes chunk of users.  If the callback is set, it will be called
// with the users slice.
func (u *Users) Users(ctx context.Context, users []slack.User) error {
	if err := u.dirproc.Users(ctx, users); err != nil {
		return err
	}
	if u.cb != nil {
		if err := u.cb(users); err != nil {
			return fmt.Errorf("users callback returned an error: %w", err)
		}
	}
	return nil
}
