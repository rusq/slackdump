package fsadapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectory_Create(t *testing.T) {
	tmpdir := t.TempDir()
	type fields struct {
		dir string
	}
	type args struct {
		fpath string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		testData []byte
		wantErr  bool
	}{
		{
			"ensure file is created and data is written (root dir)",
			fields{dir: tmpdir},
			args{"testfile.txt"},
			[]byte("123"),
			false,
		},
		{
			"ensure file is created and data is written (subdir)",
			fields{dir: tmpdir},
			args{filepath.Join("ooooh", "testfile.txt")},
			[]byte("123"),
			false,
		},
		{
			"directory (error)",
			fields{dir: tmpdir},
			args{""},
			[]byte("123"),
			true,
		},
		{
			"invalid filename",
			fields{dir: tmpdir},
			args{".."},
			[]byte("123"),
			true,
		},
		{
			"outside of root directory",
			fields{dir: tmpdir},
			args{strings.Repeat(".."+string(filepath.Separator), 20) + filepath.Join("tmp", "hello_rootfs.txt")},
			[]byte("123"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := Directory{
				dir: tt.fields.dir,
			}
			f, err := fs.Create(tt.args.fpath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			assert, require := assert.New(t), require.New(t)

			n, err := f.Write(tt.testData)
			require.NoError(err)
			assert.Equal(3, n)

			assert.NoError(f.Close())

			testFile := filepath.Join(tt.fields.dir, tt.args.fpath)
			assert.FileExists(testFile)

			fileData, err := os.ReadFile(testFile)
			assert.NoError(err)

			assert.Equal(tt.testData, fileData)
		})
	}
}

func TestDirectory_ensureSubdir(t *testing.T) {
	type fields struct {
		dir string
	}
	type args struct {
		node string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"is a subdir",
			fields{dir: filepath.Join("a", "b")},
			args{filepath.Join("a", "b", "c", "d", "e")},
			false,
		},
		{
			"is not a subdir",
			fields{dir: filepath.Join("a", "b", "d")},
			args{filepath.Join("a", "b", "c")},
			true,
		},
		{
			"path hack",
			fields{dir: filepath.Join("a", "b", "d")},
			args{filepath.Join("..", "..", "..", "tmp")},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := Directory{
				dir: tt.fields.dir,
			}
			if err := fs.ensureSubdir(tt.args.node); (err != nil) != tt.wantErr {
				t.Errorf("Directory.ensureSubdir() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_mkdirAll(t *testing.T) {
	tmpdir := t.TempDir()
	existingFile := filepath.Join(tmpdir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("123"), 0640); err != nil {
		t.Fatal(err)
	}

	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"ok",
			args{filepath.Join(tmpdir, "abc")},
			false,
		},
		{
			"empty dir - error",
			args{""},
			true,
		},
		{
			"already exists",
			args{tmpdir},
			false,
		},
		{
			"is a file",
			args{existingFile},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mkdirAll(tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("mkdirAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDirectory_WriteFile(t *testing.T) {
	tmpdir := t.TempDir()
	type fields struct {
		dir string
	}
	type args struct {
		name string
		data []byte
		perm os.FileMode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"all ok",
			fields{dir: tmpdir},
			args{"blah.txt", []byte("blah"), 0640},
			false,
		},
		{
			"outside of base path is an error",
			fields{dir: tmpdir},
			args{filepath.Join("..", "blah.txt"), []byte("blah"), 0640},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := Directory{
				dir: tt.fields.dir,
			}
			err := fs.WriteFile(tt.args.name, tt.args.data, tt.args.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.WriteFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			// verify file contents
			data, err := os.ReadFile(filepath.Join(fs.dir, tt.args.name))
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.args.data, data)
		})
	}
}
