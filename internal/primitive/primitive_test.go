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
	"reflect"
	"testing"
)

func TestIfTrue(t *testing.T) {
	type args[T any] struct {
		cond bool
		t    T
		f    T
	}
	tests := []struct {
		name string
		args args[int]
		want int
	}{
		{
			"returns true",
			args[int]{true, 1, 0},
			1,
		},
		{
			"returns false",
			args[int]{false, 1, 0},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IfTrue(tt.args.cond, tt.args.t, tt.args.f); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("iftrue() = %v, want %v", got, tt.want)
			}
		})
	}
}
