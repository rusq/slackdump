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
	"io"
	"testing"

	"github.com/rusq/slack"
)

func TestIsSlackResponseError(t *testing.T) {
	type args struct {
		e error
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"test error",
			args{
				slack.SlackErrorResponse{
					Err: "test error",
				},
				"test error",
			},
			true,
		},
		{
			"different error text",
			args{
				slack.SlackErrorResponse{
					Err: "another error",
				},
				"test error",
			},
			false,
		},
		{
			"different error",
			args{
				io.EOF,
				"test error",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSlackResponseError(tt.args.e, tt.args.s); got != tt.want {
				t.Errorf("IsSlackResponseError() = %v, want %v", got, tt.want)
			}
		})
	}
}
