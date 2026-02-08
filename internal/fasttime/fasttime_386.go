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
//go:build 386

package fasttime

import (
	"fmt"
	"strconv"
	"strings"
)

// int size on the 32-bit systems is 32 bit (surprise), this constraints us to slower 64-bit implementation.

// TS2int converts a slack timestamp to an int64 by stripping the dot and
// converting the string to an int64.  It is useful for fast comparison.
func TS2int(ts string) (int64, error) {
	if ts == "" {
		return 0, nil
	}
	i := strings.IndexByte(ts, '.')
	if i == -1 {
		return 0, fmt.Errorf("%w: %q", ErrNotATimestamp, ts)
	}
	val, err := strconv.ParseInt(ts[:i]+ts[i+1:], 10, 64)
	return int64(val), err
}
