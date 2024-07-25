package osext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSame(t *testing.T) {
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
			got, err := Same(tt.args.path1, tt.args.path2)
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
