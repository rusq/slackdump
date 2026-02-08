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
package cache

import (
	"io"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/types"
)

func ReadUsers(r io.Reader) (types.Users, error) {
	uu, err := read[slack.User](r)
	if err != nil {
		return nil, err
	}
	return types.Users(uu), nil
}

// loadUsers tries to load the users from the file. If the file does not exist
// or is older than maxAge, it returns an error.
func (m *Manager) loadUsers(dirname, filename string, suffix string, maxAge time.Duration) (types.Users, error) {
	uu, err := load[slack.User](dirname, filename, suffix, maxAge, m.createOpener())
	if err != nil {
		return nil, err
	}
	return types.Users(uu), nil
}

// saveUsers saves the users to a file, naming the file based on the filename
// and the suffix. The file will be saved in the cache directory.
func (m *Manager) saveUsers(dirname, filename string, suffix string, uu types.Users) error {
	return save(dirname, filename, suffix, []slack.User(uu), m.createOpener())
}
