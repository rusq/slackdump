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
)

// Channels keeps slice of channels.
type Channels []slack.Channel

// UserIDs returns a slice of user IDs.
func (c Channels) UserIDs() []string {
	var seen = make(map[string]bool, len(c))
	for _, m := range c {
		if m.User == "" {
			if seen[m.Creator] {
				continue
			}
			seen[m.Creator] = true
		}
		if seen[m.User] {
			continue
		}
		seen[m.User] = true
	}
	return toslice(seen)
}
