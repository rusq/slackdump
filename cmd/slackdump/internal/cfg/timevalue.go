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
package cfg

import (
	"flag"
	"time"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// TimeValue satisfies flag.Value, used for command line parsing.
type TimeValue time.Time

var _ flag.Value = &TimeValue{}

func (tv TimeValue) String() string {
	t := time.Time(tv)
	if t.IsZero() {
		return ""
	}
	if t.Truncate(24 * time.Hour).Equal(t) {
		return t.Format(structures.DateLayout)
	}
	return t.Format(structures.TimeLayout)
}

func (tv *TimeValue) Set(s string) error {
	if s == "" {
		*tv = TimeValue(time.Time{})
		return nil
	}
	if t, err := structures.TimeParse(s); err != nil {
		return err
	} else {
		*tv = TimeValue(t)
	}
	return nil
}
