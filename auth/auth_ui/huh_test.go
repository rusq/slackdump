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

package auth_ui

import "testing"

func Test_valSixDigits(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"empty",
			args{""},
			true,
		},
		{
			"too short",
			args{"12345"},
			true,
		},
		{
			"too long",
			args{"1234567"},
			true,
		},
		{
			"not a number",
			args{"123456a"},
			true,
		},
		{
			"valid",
			args{"123456"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := valSixDigits(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("valSixDigits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
