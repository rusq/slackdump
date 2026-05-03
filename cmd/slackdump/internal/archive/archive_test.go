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

package archive

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/convert/transform/fileproc"
)

func TestNewDirectory(t *testing.T) {
	t.Run("creates a directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		cd, err := NewDirectory(tmpdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cd == nil {
			t.Fatal("expected a directory, got nil")
		}
		defer cd.Close()
		assert.Equal(t, tmpdir, cd.Name())
	})
}

func TestDBControllerOptions(t *testing.T) {
	var opts dbControllerOptions

	WithDatabaseOptions(dbase.WithVerbose(true))(&opts)
	WithDatabaseOptions(dbase.WithOnlyNewOrChangedUsers(true))(&opts)
	WithFileDeduplication()(&opts)
	WithFileDeduplication()(&opts)

	assert.Len(t, opts.dbaseOptions, 2)
	assert.True(t, opts.fileDeduplicate)
}

func TestDBControllerFileDeduplicationOption(t *testing.T) {
	tests := []struct {
		name string
		opts dbControllerOptions
		want func(t *testing.T, got any)
	}{
		{
			name: "default file processor",
			want: func(t *testing.T, got any) {
				_, ok := got.(fileproc.FileProcessor)
				assert.True(t, ok)
			},
		},
		{
			name: "deduplicating file processor",
			opts: dbControllerOptions{fileDeduplicate: true},
			want: func(t *testing.T, got any) {
				_, ok := got.(*fileproc.DeduplicatingFileProcessor)
				assert.True(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dbControllerFiler(fileproc.NoopDownloader{}, nil, nil, tt.opts)
			tt.want(t, got)
		})
	}
}
