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

package apiconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_maybeAppendExt(t *testing.T) {
	type args struct {
		filename string
		ext      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"appended",
			args{"filename", ".ext"},
			"filename.ext",
		},
		{
			"empty ext",
			args{"no_ext_here", ""},
			"no_ext_here",
		},
		{
			"dot is prepended to ext",
			args{"foo", "bar"},
			"foo.bar",
		},
		{
			"same ext",
			args{"foo.bar", ".bar"},
			"foo.bar",
		},
		{
			"already has an extension",
			args{"filename.xxx", ".ext"},
			"filename.xxx.ext",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maybeAppendExt(tt.args.filename, tt.args.ext); got != tt.want {
				t.Errorf("maybeAppendExt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_maybeFixExt(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"already toml",
			args{filename: "lol.toml"},
			"lol.toml",
		},
		{
			"already tml",
			args{filename: "lol.tml"},
			"lol.tml",
		},
		{
			"no extension",
			args{filename: "foo"},
			"foo.toml",
		},
		{
			"different extension",
			args{filename: "foo.bar"},
			"foo.bar.toml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maybeFixExt(tt.args.filename); got != tt.want {
				t.Errorf("maybeFixExt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runConfigNew(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "test.toml")
	if err := os.MkdirAll(existingDir, 0777); err != nil {
		t.Fatal(err)
	}
	type args struct {
		args []string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		shouldExist bool
	}{
		{
			"no arguments given",
			args{},
			true,
			false,
		},
		{
			"file is created",
			args{[]string{filepath.Join(dir, "sample.tml")}},
			false,
			true,
		},
		{
			"directory test.toml",
			args{[]string{existingDir}},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runConfigNew(t.Context(), CmdConfigNew, tt.args.args); (err != nil) != tt.wantErr {
				t.Errorf("runConfigNew() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(tt.args.args) == 0 {
				return
			}
			_, err := os.Stat(tt.args.args[0])
			if (err == nil) != tt.shouldExist {
				t.Errorf("file exist error: %s, shouldExist = %v", err, tt.shouldExist)
			}
		})
	}
}
