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
package directory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

func makeTestDir(t *testing.T) *chunk.Directory {
	t.Helper()

	dir := t.TempDir()
	cd, err := chunk.CreateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cd
}

func Test_filetracker_create(t *testing.T) {
	t.Parallel()
	t.Run("created a new file", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		defer tr.CloseAll()
		id := chunk.FileID("test")
		err := tr.create(id)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := tr.files[id]; !ok {
			t.Error("file not created")
		}
		if _, err := cd.Stat(id); err != nil {
			t.Fatalf("stat error: %s", err)
		}
		assert.Equal(t, 1, tr.RefCount(id), "reference count mismatch")
	})
	t.Run("does not attempt to create an existing file", func(t *testing.T) {
		t.Parallel()
		// creating initial file.
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		defer tr.CloseAll()
		id := chunk.FileID("test")
		if err := tr.create(id); err != nil {
			t.Fatal(err)
		}

		// creating again should not return an error.
		if err := tr.create(id); err != nil {
			t.Fatal(err)
		}

		if _, ok := tr.files[id]; !ok {
			t.Error("file not created")
		}
		if _, err := cd.Stat(id); err != nil {
			t.Fatalf("stat error: %s", err)
		}
		assert.Equal(t, 1, tr.RefCount(id), "reference count mismatch")
	})
}

func Test_filetracker_Recorder(t *testing.T) {
	t.Parallel()
	t.Run("returns existing processor", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		id := chunk.FileID("test")
		if err := tr.create(id); err != nil {
			t.Fatal(err)
		}
		r, err := tr.Recorder(id)
		if err != nil {
			t.Fatal(err)
		}
		if r == nil {
			t.Fatal("nil processor")
		}
		r.Close()
	})
	t.Run("creates new processor", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		id := chunk.FileID("test")
		r, err := tr.Recorder(id)
		if err != nil {
			t.Fatal(err)
		}
		if r == nil {
			t.Fatal("nil processor")
		}
		r.Close()
	})
	t.Run("returns another processor for different file", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		id1 := chunk.FileID("test1")
		id2 := chunk.FileID("test2")
		r1, err := tr.Recorder(id1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cd.Stat(id1); err != nil {
			t.Fatal(err)
		}
		r2, err := tr.Recorder(id2)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cd.Stat(id2); err != nil {
			t.Fatal(err)
		}
		if r1 == nil || r2 == nil {
			t.Fatal("nil processor")
		}
		r1.Close()
		r2.Close()
	})
}

func Test_filetracker_CloseAll(t *testing.T) {
	t.Run("closes open files", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		id1 := chunk.FileID("test1")
		id2 := chunk.FileID("test2")

		if _, err := tr.Recorder(id1); err != nil {
			t.Fatal(err)
		}

		if _, err := tr.Recorder(id2); err != nil {
			t.Fatal(err)
		}
		if err := tr.CloseAll(); err != nil {
			t.Fatal(err)
		}
		if len(tr.files) != 0 {
			t.Error("files not closed")
		}
	})
	t.Run("does nothing if there's no files", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		if err := tr.CloseAll(); err != nil {
			t.Fatal(err)
		}
	})
}

func Test_filetracker_RefCount(t *testing.T) {
	t.Run("returns reference count", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		id := chunk.FileID("test")
		r, err := tr.Recorder(id)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		assert.Equal(t, 1, tr.RefCount(id), "reference count mismatch")
	})
	t.Run("returns 0 for non-existing file", func(t *testing.T) {
		t.Parallel()
		cd := makeTestDir(t)
		tr := newFileTracker(cd)
		assert.Equal(t, 0, tr.RefCount(chunk.FileID("test")), "reference count mismatch")
	})
}

func Test_filetracker_unregister(t *testing.T) {
	// create a test file.
	cd := makeTestDir(t)
	tr := newFileTracker(cd)
	testID := chunk.FileID("test")
	if err := tr.create(testID); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		dir   *chunk.Directory
		files map[chunk.FileID]*entityproc
	}
	type args struct {
		id chunk.FileID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"unregister non-existing file",
			fields{
				dir:   makeTestDir(t),
				files: make(map[chunk.FileID]*entityproc),
			},
			args{
				id: chunk.FileID("test"),
			},
			false,
		},
		{
			"unregister existing file",
			fields{
				dir:   tr.dir,
				files: tr.files,
			},
			args{
				id: testID,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &filetracker{
				dir:   tt.fields.dir,
				files: tt.fields.files,
			}
			if err := tr.unregister(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("filetracker.unregister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
