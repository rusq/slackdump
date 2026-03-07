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

package client

import (
	"testing"
)

func Test_roundRobin_next(t *testing.T) {
	type fields struct {
		total int
		i     int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "test1",
			fields: fields{total: 3, i: 0},
			want:   1,
		},
		{
			name:   "test2",
			fields: fields{total: 3, i: 1},
			want:   2,
		},
		{
			name:   "test3",
			fields: fields{total: 3, i: 2},
			want:   0,
		},
		{
			name:   "test4",
			fields: fields{total: 2, i: 0},
			want:   1,
		},
		{
			name:   "test5",
			fields: fields{total: 2, i: 1},
			want:   0,
		},
		{
			name:   "test6",
			fields: fields{total: 1, i: 0},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &roundRobin{
				total: tt.fields.total,
				i:     tt.fields.i,
			}
			if got := r.next(); got != tt.want {
				t.Errorf("roundRobin.next() = %v, want %v", got, tt.want)
			}
		})
	}
}
