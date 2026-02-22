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

package updater

import (
	"testing"
	"time"
)

func TestRelease_Equal(t *testing.T) {
	baseTime := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	differentTime := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		r         Release
		other     Release
		wantEqual bool
	}{
		{
			name: "identical releases",
			r: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			wantEqual: true,
		},
		{
			name: "same version case insensitive",
			r: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "V3.0.0",
				PublishedAt: baseTime,
			},
			wantEqual: true,
		},
		{
			name: "same version but different published dates",
			r: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "v3.0.0",
				PublishedAt: differentTime,
			},
			wantEqual: false,
		},
		{
			name: "different versions",
			r: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "v3.1.0",
				PublishedAt: baseTime,
			},
			wantEqual: false,
		},
		{
			name: "different versions and dates",
			r: Release{
				Version:     "v3.0.0",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "v3.1.0",
				PublishedAt: differentTime,
			},
			wantEqual: false,
		},
		{
			name: "empty versions",
			r: Release{
				Version:     "",
				PublishedAt: baseTime,
			},
			other: Release{
				Version:     "",
				PublishedAt: baseTime,
			},
			wantEqual: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEqual := tt.r.Equal(tt.other)
			if gotEqual != tt.wantEqual {
				t.Errorf("Release.Equal() = %v, want %v", gotEqual, tt.wantEqual)
			}
		})
	}
}
