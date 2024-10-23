package filemgr

import (
	"bytes"
	"io/fs"
	"reflect"
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
			want: "No files found, press [Backspace]\n\n ↑↓ move•[⏎] select•[⇤] back•[q] quit\n",
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
