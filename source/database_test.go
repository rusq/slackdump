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
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
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

// aliasWriter is a subset of viewer.Aliaser covering only the write methods.
// Defined here to avoid an import cycle with the viewer package.
type aliasWriter interface {
	SetAlias(id, alias string) error
	DeleteAlias(id string) error
}

func TestOpenDatabaseRW_writable(t *testing.T) {
	dbpath := filepath.Join(fixturesDir, "source_database.db")
	got, err := OpenDatabaseRW(t.Context(), dbpath)
	if err != nil {
		t.Fatalf("OpenDatabaseRW() error = %v", err)
	}
	defer got.(io.Closer).Close()

	if _, ok := got.(*RWDatabase); !ok {
		t.Errorf("OpenDatabaseRW() = %T, want *RWDatabase", got)
	}
	if _, ok := got.(aliasWriter); !ok {
		t.Error("OpenDatabaseRW() result does not implement aliasWriter (Aliaser)")
	}
}

func TestOpenDatabaseRW_fallback(t *testing.T) {
	// Replace the rw-open function so it fails, exercising the ro fallback.
	orig := openRWFn
	t.Cleanup(func() { openRWFn = orig })
	openRWFn = func(_ context.Context, _ string) (*dbase.RWSource, error) {
		return nil, errors.New("simulated rw open failure")
	}

	dbpath := filepath.Join(fixturesDir, "source_database.db")
	got, err := OpenDatabaseRW(t.Context(), dbpath)
	if err != nil {
		t.Fatalf("OpenDatabaseRW() fallback error = %v", err)
	}
	defer got.(io.Closer).Close()

	if _, ok := got.(*Database); !ok {
		t.Errorf("OpenDatabaseRW() fallback = %T, want *Database", got)
	}
	if _, ok := got.(aliasWriter); ok {
		t.Error("OpenDatabaseRW() fallback result unexpectedly implements aliasWriter (Aliaser)")
	}
}
