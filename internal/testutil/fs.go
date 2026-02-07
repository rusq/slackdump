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
package testutil

import (
	"io/fs"
	"os"
	"testing"
)

type FileInfo struct {
	Name string
	Size int64
}

// CollectFiles returns a map of file paths to file info.
func CollectFiles(t *testing.T, fsys fs.FS) (ret map[string]FileInfo) {
	t.Helper()
	ret = make(map[string]FileInfo)
	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		ret[path] = FileInfo{Name: d.Name(), Size: fi.Size()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return
}

// PrepareTestDirectory prepares a temporary directory for testing and populates it with
// files from fsys.  It returns the path to the directory.
func PrepareTestDirectory(t *testing.T, fsys fs.FS) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, fsys); err != nil {
		t.Fatal(err)
	}
	return dir
}
