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
	"testing"

	"github.com/rusq/slackdump/v4/internal/fixtures"
)

func TestUserIndex_IsDeleted(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		idx  UserIndex
		args args
		want bool
	}{
		{
			name: "deleted",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"DELD"},
			want: true,
		},
		{
			name: "not deleted",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"LOL1"},
			want: false,
		},
		{
			name: "not present",
			idx:  NewUserIndex(fixtures.TestUsers),
			args: args{"XXX"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.idx.IsDeleted(tt.args.id); got != tt.want {
				t.Errorf("UserIndex.IsDeleted() = %v, want %v", got, tt.want)
			}
		})
	}

}

func Test_nvl(t *testing.T) {
	type args struct {
		s  string
		ss []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"returns the fist arg",
			args{"a", []string{"b", "c", "d"}},
			"a",
		},
		{
			"returns the fist arg",
			args{"", []string{"b", "c", "d"}},
			"b",
		},
		{
			"returns the fist arg",
			args{"", []string{"", "", "d"}},
			"d",
		},
		{
			"returns empty if everything is empty",
			args{"", []string{"", "", ""}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NVL(tt.args.s, tt.args.ss...); got != tt.want {
				t.Errorf("nvl() = %v, want %v", got, tt.want)
			}
		})
	}
}
