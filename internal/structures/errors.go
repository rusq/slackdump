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
	"errors"
	"strings"

	"github.com/rusq/slack"
)

// IsSlackResponseError returns true if the following conditions are met:
// - error is of [slack.SlackErrorResponse] type; AND
// - e.Err field equal to the string s.
// otherwise, returns false.
func IsSlackResponseError(e error, s string) bool {
	var se slack.SlackErrorResponse
	return errors.As(e, &se) && strings.EqualFold(se.Err, s)
}
