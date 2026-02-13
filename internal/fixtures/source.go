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

package fixtures

import (
	"archive/zip"
	"bytes"
	"embed"
	"io/fs"
	"path"
	"testing"
)

//go:embed assets/source_dump_dir
var fsTestDumpDir embed.FS

//go:embed assets/source_dump.zip
var fsTestDumpZIP []byte

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

var testDumpDirPath = path.Join("assets", "source_dump_dir")

func init() {
	FSTestDumpDir = must(fs.Sub(fsTestDumpDir, testDumpDirPath))
}

// FSTempDumpDir is a filesystem with a dump directory.
var FSTestDumpDir fs.FS

// FSTestDumpZIP returns a filesystem of Dump ZIP archive.
func FSTestDumpZIP(t *testing.T) fs.FS {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(fsTestDumpZIP), int64(len(fsTestDumpZIP)))
	if err != nil {
		t.Fatal(err)
	}
	return zr
}
