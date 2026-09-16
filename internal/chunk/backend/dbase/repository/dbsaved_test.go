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
	"reflect"
	"testing"

	"github.com/rusq/slackdump/v4/internal/edge"
)

var savedItem1 = &edge.SavedItem{
	ItemID:    "C123",
	ItemType:  "message",
	Timestamp: "1725318212.603879",
	State:     "in_progress",
	TodoState: "saved",
}

func TestNewDBSavedItem(t *testing.T) {
	type args struct {
		chunkID int64
		idx     int
		si      *edge.SavedItem
	}
	tests := []struct {
		name    string
		args    args
		want    *DBSavedItem
		wantErr bool
	}{
		{
			name: "creates a new DBSavedItem",
			args: args{
				chunkID: 42,
				idx:     50,
				si:      savedItem1,
			},
			want: &DBSavedItem{
				ID:        0, // autoincrement, handled by the database.
				ChunkID:   42,
				ItemID:    "C123",
				ItemType:  "message",
				TS:        new("1725318212.603879"),
				State:     new("in_progress"),
				TodoState: new("saved"),
				IDX:       50,
				Data:      must(marshal(savedItem1)),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDBSavedItem(tt.args.chunkID, tt.args.idx, tt.args.si)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBSavedItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDBSavedItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
