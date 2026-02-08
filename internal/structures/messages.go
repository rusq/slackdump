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
package structures

import (
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/fasttime"
)

type Messages []slack.Message

func (m Messages) Len() int { return len(m) }
func (m Messages) Less(i, j int) bool {
	tsi, err := fasttime.TS2int(m[i].Timestamp)
	if err != nil {
		return false
	}
	tsj, err := fasttime.TS2int(m[j].Timestamp)
	if err != nil {
		return false
	}
	return tsi < tsj
}
func (m Messages) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
