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

import (
	"fmt"
	"iter"
)

// Collect collects all Ks from iterator it, returning any encountered error.
func Collect[K any](it iter.Seq2[K, error]) ([]K, error) {
	kk := make([]K, 0)
	for k, err := range it {
		if err != nil {
			return kk, fmt.Errorf("iterator error: %w", err)
		}
		kk = append(kk, k)
	}
	return kk, nil
}
