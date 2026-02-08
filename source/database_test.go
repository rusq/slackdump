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

	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
)

func TestDatabase_Name(t *testing.T) {
	type fields struct {
		name    string
		files   Storage
		avatars Storage
		Source  *dbase.Source
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "test",
			fields: fields{
				name:    "foobar",
				files:   NoStorage{},
				avatars: NoStorage{},
			},
			want: "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Database{
				name:    tt.fields.name,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
				Source:  tt.fields.Source,
			}
			if got := d.Name(); got != tt.want {
				t.Errorf("Database.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDatabase_Type(t *testing.T) {
	type fields struct {
		name    string
		files   Storage
		avatars Storage
		Source  *dbase.Source
	}
	tests := []struct {
		name   string
		fields fields
		want   Flags
	}{
		{
			name: "test",
			want: FDatabase,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Database{
				name:    tt.fields.name,
				files:   tt.fields.files,
				avatars: tt.fields.avatars,
				Source:  tt.fields.Source,
			}
			if got := d.Type(); got != tt.want {
				t.Errorf("Database.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
