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
	_ "embed"
	"testing"
)

func TestStripZipExt(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"zip",
			args{"foo.zip"},
			"foo",
		},
		{
			"tar.gz",
			args{"foo.tar.gz"},
			"foo.tar.gz",
		},
		{
			"ZIP",
			args{"foo.ZIP"},
			"foo",
		},
		{
			"empty",
			args{""},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripZipExt(tt.args.s); got != tt.want {
				t.Errorf("StripZipExt() = %v, want %v", got, tt.want)
			}
		})
	}
}
