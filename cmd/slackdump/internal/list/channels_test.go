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
package list

import (
	"testing"

	"github.com/rusq/slackdump/v3/types"
)

func Test_channels_Len(t *testing.T) {
	type fields struct {
		channels types.Channels
		users    types.Users
		opts     channelOptions
		common   commonOpts
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "zero",
			want: 0,
		},
		{
			name:   "two",
			fields: fields{channels: make(types.Channels, 2)},
			want:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &channels{
				channels: tt.fields.channels,
				users:    tt.fields.users,
				opts:     tt.fields.opts,
				common:   tt.fields.common,
			}
			if got := l.Len(); got != tt.want {
				t.Errorf("channels.Len() = %v, want %v", got, tt.want)
			}
		})
	}
}
