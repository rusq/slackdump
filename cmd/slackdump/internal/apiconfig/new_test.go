package apiconfig

import (
	"context"
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
			"already yaml",
			args{filename: "lol.yaml"},
			"lol.yaml",
		},
		{
			"already yml",
			args{filename: "lol.yml"},
			"lol.yml",
		},
		{
			"no extension",
			args{filename: "foo"},
			"foo.yaml",
		},
		{
			"different extension",
			args{filename: "foo.bar"},
			"foo.bar.yaml",
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

func Test_shouldOverwrite(t *testing.T) {
	dir := t.TempDir()
	existingFile, err := os.CreateTemp(dir, "unittest*")
	if err != nil {
		t.Fatal(err)
	}
	defer existingFile.Close()

	existingDir := filepath.Join(dir, "existing_dir")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatal(err)
	}
	type args struct {
		filename string
		override bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"non-existing file",
			args{"$$$$", false},
			true,
		},
		{
			"non-existing file override",
			args{"$$$$", true},
			true,
		},
		{
			"existing file",
			args{existingFile.Name(), false},
			false,
		},
		{
			"existing file override",
			args{existingFile.Name(), true},
			true,
		},
		{
			"existing directory",
			args{existingDir, false},
			false,
		},
		{
			"existing directory override",
			args{existingDir, true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldOverwrite(tt.args.filename, tt.args.override); got != tt.want {
				t.Errorf("shouldOverwrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runConfigNew(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "test.yaml")
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
			args{[]string{filepath.Join(dir, "sample.yml")}},
			false,
			true,
		},
		{
			"directory test.yaml",
			args{[]string{existingDir}},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := runConfigNew(context.Background(), CmdConfigNew, tt.args.args); (err != nil) != tt.wantErr {
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
