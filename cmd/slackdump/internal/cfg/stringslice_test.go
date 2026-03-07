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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSlice_Set(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		ss   *StringSlice
		args args
		want StringSlice
	}{
		{
			name: "sets the string slice",
			ss:   new(StringSlice),
			args: args{"alpha,bravo,charlie"},
			want: StringSlice{"alpha", "bravo", "charlie"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ss.Set(tt.args.s)
			assert.Equal(t, tt.want, *tt.ss)
		})
	}
}

func TestStringSlice_String(t *testing.T) {
	tests := []struct {
		name string
		ss   *StringSlice
		want string
	}{
		{
			name: "abc",
			ss:   &StringSlice{"alpha", "bravo", "charlie"},
			want: "alpha,bravo,charlie",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ss.String(); got != tt.want {
				t.Errorf("StringSlice.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
