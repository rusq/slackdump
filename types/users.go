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

package types

import (
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// Users is a slice of users.
type Users []slack.User

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() structures.UserIndex {
	return structures.NewUserIndex(us)
}

// UserIDs returns a slice of user IDs.
func (us Users) UserIDs() []string {
	var ids = make([]string, len(us))
	for i := range us {
		ids[i] = us[i].ID
	}
	return ids
}
