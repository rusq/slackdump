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

package filemgr

import (
	"bytes"
	"io/fs"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/rusq/rbubbles/display"
	"github.com/stretchr/testify/assert"
)

var testfs = fstest.MapFS{
	"dir1": &fstest.MapFile{
		Mode: fs.ModeDir,
	},
	"dir2": &fstest.MapFile{
		Mode: fs.ModeDir,
	},
	"dir2/dirfile.txt": &fstest.MapFile{
		Data: []byte("dir2/dirfile.txt"),
	},
	"file1.txt": &fstest.MapFile{
		Data: []byte("file1"),
	},
	"file2.txt": &fstest.MapFile{
		Data: []byte("file2"),
	},
	"file3.txt": &fstest.MapFile{
		Data: []byte("file3"),
	},
	"binary1.bin": &fstest.MapFile{
		Data: []byte{0x01, 0x02, 0x03},
	},
	"binary2.bin": &fstest.MapFile{
		Data: []byte{0x04, 0x05, 0x06},
	},
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Test_collectFiles(t *testing.T) {
	type args struct {
		fsys  fs.FS
		globs []string
	}
	tests := []struct {
		name      string
		args      args
		wantFiles []fs.FileInfo
		wantErr   bool
	}{
		{
			name: "collect all files",
			args: args{
				fsys:  testfs,
				globs: []string{"*"},
			},
			wantFiles: []fs.FileInfo{
				must(fs.Stat(testfs, "binary1.bin")),
				must(fs.Stat(testfs, "binary2.bin")),
				must(fs.Stat(testfs, "file1.txt")),
				must(fs.Stat(testfs, "file2.txt")),
				must(fs.Stat(testfs, "file3.txt")),
			},
			wantErr: false,
		},
		{
			name: "collect only binary files",
			args: args{
				fsys:  testfs,
				globs: []string{"*.bin"},
			},
			wantFiles: []fs.FileInfo{
				must(fs.Stat(testfs, "binary1.bin")),
				must(fs.Stat(testfs, "binary2.bin")),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFiles, err := collectFiles(tt.args.fsys, tt.args.globs...)
			if (err != nil) != tt.wantErr {
				t.Errorf("collectFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Len(t, gotFiles, len(tt.wantFiles))
			assert.True(t, slices.EqualFunc(tt.wantFiles, gotFiles, func(a, b fs.FileInfo) bool {
				t.Logf("%s, %s => %v", a.Name(), b.Name(), a.Name() == b.Name())
				return a.Name() == b.Name()
			}))
		})
	}
}

func Test_collectDirs(t *testing.T) {
	type args struct {
		fsys fs.FS
	}
	tests := []struct {
		name    string
		args    args
		want    []fs.FileInfo
		wantErr bool
	}{
		{
			name: "collect all dirs",
			args: args{
				fsys: testfs,
			},
			want: []fs.FileInfo{
				must(fs.Stat(testfs, "dir1")),
				must(fs.Stat(testfs, "dir2")),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := collectDirs(tt.args.fsys)
			if (err != nil) != tt.wantErr {
				t.Errorf("collectDirs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("collectDirs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_View(t *testing.T) {
	allFiles := func(t *testing.T, sub string) []fs.FileInfo {
		t.Helper()
		if sub == "" {
			sub = "."
		}
		msg, err := readFS(testfs, sub, "*")
		if err != nil {
			t.Fatal(err)
		}
		return msg.files
	}

	type fields struct {
		Globs     []string
		Selected  string
		FS        fs.FS
		Directory string
		Height    int
		ShowHelp  bool
		Style     Style
		files     []fs.FileInfo
		finished  bool
		st        display.State
		viewStack display.Stack[display.State]
		Debug     bool
		last      string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "view",
			fields: fields{
				Globs:     []string{"*"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: ".",
				Height:    len(testfs),
				st: display.State{
					Max: len(testfs),
				},
				files: allFiles(t, "."),
			},
			want: "binary1.bin         3B 01-01-0001 00:00\nbinary2.bin         3B 01-01-0001 00:00\nfile1.txt           5B 01-01-0001 00:00\nfile2.txt           5B 01-01-0001 00:00\nfile3.txt           5B 01-01-0001 00:00\ndir1             <DIR> 01-01-0001 00:00\ndir2             <DIR> 01-01-0001 00:00\n                                       \n",
		},
		{
			name: "finished",
			fields: fields{
				Globs:     []string{"*"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: ".",
				Height:    len(testfs),
				st: display.State{
					Max: len(testfs),
				},
				files:    allFiles(t, "."),
				finished: true, // finished!
			},
			want: "",
		},
		{
			name: "subdir",
			fields: fields{
				Globs:     []string{"*"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: "dir2",
				Height:    len(testfs),
				st: display.State{
					Max: len(testfs),
				},
				files: allFiles(t, "dir2"),
			},
			want: "..               <DIR> 01-01-0001 00:00\ndirfile.txt        16B 01-01-0001 00:00\n                                       \n                                       \n                                       \n                                       \n                                       \n                                       \n",
		},
		{
			name: "no files found",
			fields: fields{
				Globs:     []string{"*.foo"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: ".",
				Height:    2,
				st: display.State{
					Max: 2,
				},
				files: []fs.FileInfo{},
			},
			want: "No files found, press [Backspace]\n                                       \n",
		},
		{
			name: "no files with help",
			fields: fields{
				Globs:     []string{"*.foo"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: ".",
				Height:    1,
				st: display.State{
					Max: 1,
				},
				files:    []fs.FileInfo{},
				ShowHelp: true,
			},
			want: "No files found, press [Backspace]\n\n ↑↓ move•[⏎] select•[⇤] back•[q] quit",
		},
		{
			name: "window height less than number of files",
			fields: fields{
				Globs:     []string{"*"},
				Selected:  "file1.txt",
				FS:        testfs,
				Directory: ".",
				Height:    2,
				st: display.State{
					Max: 1,
				},
				files: allFiles(t, "."),
			},
			want: "binary1.bin         3B 01-01-0001 00:00\nbinary2.bin         3B 01-01-0001 00:00\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				Globs:     tt.fields.Globs,
				Selected:  tt.fields.Selected,
				FS:        tt.fields.FS,
				Directory: tt.fields.Directory,
				Height:    tt.fields.Height,
				ShowHelp:  tt.fields.ShowHelp,
				Style:     tt.fields.Style,
				files:     tt.fields.files,
				finished:  tt.fields.finished,
				st:        tt.fields.st,
				viewStack: tt.fields.viewStack,
				Debug:     tt.fields.Debug,
				last:      tt.fields.last,
			}
			assert.Equal(t, tt.want, m.View())
		})
	}
}

func Test_humanizeSize(t *testing.T) {
	type args struct {
		size int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "bytes",
			args: args{
				size: 3,
			},
			want: "    3B",
		},
		{
			name: "kilobytes",
			args: args{
				size: 1024,
			},
			want: "  1.0K",
		},
		{
			name: "megabytes",
			args: args{
				size: 1024 * 1024,
			},
			want: "  1.0M",
		},
		{
			name: "gigabytes",
			args: args{
				size: 1024 * 1024 * 1024,
			},
			want: "  1.0G",
		},
		{
			name: "terabytes",
			args: args{
				size: 1024 * 1024 * 1024 * 1024,
			},
			want: "  1.0T",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanizeSize(tt.args.size); got != tt.want {
				t.Errorf("humanizeSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModel_printDebug(t *testing.T) {
	type fields struct {
		Globs     []string
		Selected  string
		FS        fs.FS
		Directory string
		Height    int
		ShowHelp  bool
		Style     Style
		files     []fs.FileInfo
		finished  bool
		st        display.State
		viewStack display.Stack[display.State]
		Debug     bool
		last      string
	}
	tests := []struct {
		name   string
		fields fields
		wantW  string
	}{
		{
			name:   "debug",
			fields: fields{},
			wantW:  "cursor: 0\nmin: 0\nmax: 0\nlast: \"\"\ndir: \"\"\nselected: \"\"\n|123456789|123456789|123456789|123456789\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				Globs:     tt.fields.Globs,
				Selected:  tt.fields.Selected,
				FS:        tt.fields.FS,
				Directory: tt.fields.Directory,
				Height:    tt.fields.Height,
				ShowHelp:  tt.fields.ShowHelp,
				Style:     tt.fields.Style,
				files:     tt.fields.files,
				finished:  tt.fields.finished,
				st:        tt.fields.st,
				viewStack: tt.fields.viewStack,
				Debug:     tt.fields.Debug,
				last:      tt.fields.last,
			}
			w := &bytes.Buffer{}
			m.printDebug(w)
			assert.Equal(t, tt.wantW, w.String())
		})
	}
}

func TestModel_shorten(t *testing.T) {
	type args struct {
		dirpath string
	}
	tests := []struct {
		name    string
		windows bool
		args    args
		want    string
	}{
		{
			name: "very short path",
			args: args{
				dirpath: "/",
			},
			want: "/",
		},
		{
			name: "longer path",
			args: args{
				dirpath: "/home/user/Downloads/Funky/Long/Path/Longer/Than/40/Characters",
			},
			want: "/h/u/D/F/L/P/L/T/4/Characters",
		},
		{
			name: "really long path",
			args: args{
				dirpath: "/home/user/Downloads/Funky/Long/Path/Longer/Than/40/Characters/And/Even/Longer/Than/That/And/Then/Some/More/And/Even/Longer/Than/That/And/Then/Some",
			},
			want: "…/A/E/L/T/T/A/T/S/M/A/E/L/T/T/A/T/Some",
		},
		{
			name:    "windows",
			windows: true,
			args: args{
				dirpath: "D:\\Users\\User\\Downloads",
			},
			want: "D:\\Users\\User\\Downloads",
		},
		{
			name:    "very long windows path",
			windows: true,
			args: args{
				dirpath: "C:\\Program Files\\Microsoft Visual Studio\\2022\\Community\\Some Funky\\Path That\\Nobody In Sane\\Mind Can\\Remember\\Or Type\\Without Making\\Over 9000\\Typos",
			},
			want: "C:\\P\\M\\2\\C\\S\\P\\N\\M\\R\\O\\W\\O\\Typos",
		},
		{
			name:    "longer than width",
			windows: true,
			args: args{
				dirpath: "C:\\P\\M\\2\\C\\S\\P\\N\\M\\R\\O\\W\\O\\T\\S\\F\\K\\L\\M\\N\\O\\P\\Q\\R\\",
			},
			want: "…S\\P\\N\\M\\R\\O\\W\\O\\T\\S\\F\\K\\L\\M\\N\\O\\P\\Q\\R",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (runtime.GOOS == "windows") != tt.windows {
				t.Skip("skipping test on non-windows OS")
			}
			m := Model{}
			if got := m.shorten(tt.args.dirpath); got != tt.want {
				t.Errorf("Model.shorten() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toFSpath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name    string
		windows bool
		args    args
		want    string
	}{
		{
			name:    "updates path on windows",
			windows: true,
			args: args{
				p: "C:\\Program Files\\Microsoft Office 95",
			},
			want: "C:/Program Files/Microsoft Office 95",
		},
		{
			name:    "returns as is on non-windows",
			windows: false,
			args: args{
				p: "/var/spool/mail/root",
			},
			want: "/var/spool/mail/root",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (runtime.GOOS == "windows") != tt.windows {
				t.Skip("skipping")
			}
			if got := toFSpath(tt.args.p); got != tt.want {
				t.Errorf("toFSpath() = %v, want %v", got, tt.want)
			}
		})
	}
}
