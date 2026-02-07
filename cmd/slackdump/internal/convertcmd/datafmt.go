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
package convertcmd

import (
	"fmt"
	"strings"
)

// datafmt is an enumeration of supported data formats.
//
//go:generate stringer -type=datafmt -trimprefix=F
type datafmt uint8

const (
	Fdump datafmt = iota
	Fexport
	Fchunk
	Fdatabase
)

func (e *datafmt) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_datafmt_index)-1; i++ {
		if strings.ToLower(_datafmt_name[_datafmt_index[i]:_datafmt_index[i+1]]) == v {
			*e = datafmt(i)
			return nil
		}
	}
	return fmt.Errorf("unknown format: %s", v)
}
