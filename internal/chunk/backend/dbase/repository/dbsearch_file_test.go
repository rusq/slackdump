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

package repository

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

func TestNewDBSearchFile(t *testing.T) {
	type args struct {
		chunkID int64
		n       int
		sf      *slack.File
	}
	tests := []struct {
		name    string
		args    args
		want    *DBSearchFile
		wantErr bool
	}{
		{
			name: "creates a new DBSearchFile",
			args: args{chunkID: 42, n: 50, sf: file1},
			want: &DBSearchFile{
				ID:      0, // autoincrement, handled by the database.
				ChunkID: 42,
				FileID:  "FILE1",
				Index:   50,
				Data:    must(marshal(file1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBSearchFile(tt.args.chunkID, tt.args.n, tt.args.sf)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBSearchFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
