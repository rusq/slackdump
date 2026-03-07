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

package osext

import (
	"os"
	"path/filepath"
	"testing"

	fx "github.com/rusq/slackdump/v4/internal/fixtures"
)

func TestIsSame(t *testing.T) {
	fx.SkipOnWindows(t)
	baseDir := t.TempDir()

	file1 := filepath.Join(baseDir, "file1")
	file2 := filepath.Join(baseDir, "file2")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// file1rel is the path relative to the current working directory (where
	// the test is running).
	file1rel, err := filepath.Rel(wd, file1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("file1rel: %q", file1rel)

	type args struct {
		path1 string
		path2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"same file",
			args{file1, file1},
			true,
			false,
		},
		{
			"same file relative",
			args{file1, file1rel},
			true,
			false,
		},
		{
			"different files",
			args{file1, file2},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsSame(tt.args.path1, tt.args.path2)
			if (err != nil) != tt.wantErr {
				t.Errorf("Same() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Same() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	fx.SkipOnWindows(t) // symlinks
	d := t.TempDir()

	// creating fixtures
	testFile := filepath.Join(d, "file")
	fx.MkTestFileName(t, testFile, "test")

	testDir := filepath.Join(d, "dir")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// creating a symlink to the testDir
	testDirSym := filepath.Join(d, "dir-sym")
	if err := os.Symlink(testDir, testDirSym); err != nil {
		t.Fatal(err)
	}

	testFileSym := filepath.Join(d, "file-sym")
	if err := os.Symlink(testFile, testFileSym); err != nil {
		t.Fatal(err)
	}

	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"directory - ok",
			args{testDir},
			false,
		},
		{
			"directory symlink - ok",
			args{testDirSym},
			false,
		},
		{
			"file - not a directory",
			args{testFile},
			true,
		},
		{
			"file symlink - not a directory",
			args{testFileSym},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DirExists(tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("DirExists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
