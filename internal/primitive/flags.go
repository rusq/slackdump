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

package primitive

import "bytes"

// FlagRender returns the representation of f using characters in flg.
func FlagRender(f uint8, flg [8]byte) string {
	const bits = 8 - 1
	var buf bytes.Buffer
	for i := bits; i >= 0; i-- {
		if f&(1<<uint(i)) != 0 {
			buf.WriteByte(flg[bits-i])
		} else {
			buf.WriteByte('.')
		}
	}
	return buf.String()
}
