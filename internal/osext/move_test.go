package osext

import (
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"
	fx "github.com/rusq/slackdump/v3/internal/fixtures"
)

func TestMoveFile(t *testing.T) {
	d := t.TempDir()

	// fixtures

	fsa := fsadapter.NewDirectory(d)
	defer fsa.Close()

	// create source file
	srcf := filepath.Join(d, "src")
	fx.MkTestFileName(t, srcf, "test")

	type args struct {
		src string
		fs  fsadapter.FS
		dst string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"existing source file",
			args{srcf, fsa, "dst"},
			false,
		},
		{
			"non-existing source file",
			args{filepath.Join(d, "non-existing"), fsa, "dst"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := MoveFile(tt.args.src, tt.args.fs, tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("MoveFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
