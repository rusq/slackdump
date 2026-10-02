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

package downloader

import (
	"errors"
	"testing"
)

func Test_redactURLText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "url error with quoted URL",
			in:   `Get "https://user:password@example.com/file?token=secret": connection refused`,
			want: `Get "https://example.com/file": connection refused`,
		},
		{
			name: "bare URL with fragment",
			in:   "download https://example.com/file#private failed",
			want: "download https://example.com/file failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactURLText(tt.in); got != tt.want {
				t.Errorf("redactURLText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_redactError(t *testing.T) {
	err := redactError(errors.New(`Get "https://example.com/file?token=secret": connection refused`))
	if got := err.Error(); got != `Get "https://example.com/file": connection refused` {
		t.Errorf("redactError().Error() = %q", got)
	}
}
