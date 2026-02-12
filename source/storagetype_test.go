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

package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageType_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		e       *StorageType
		args    args
		want    StorageType
		wantErr bool
	}{
		{
			name: "STmattermost",
			e:    new(StorageType),
			args: args{v: "mattermost"},
			want: STmattermost,
		},
		{
			name: "STstandard",
			e:    new(StorageType),
			args: args{v: "standard"},
			want: STstandard,
		},
		{
			name: "STdump",
			e:    new(StorageType),
			args: args{v: "dump"},
			want: STdump,
		},
		{
			name:    "invalid",
			e:       new(StorageType),
			args:    args{v: "invalid"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("StorageType.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, *tt.e)
		})
	}
}
