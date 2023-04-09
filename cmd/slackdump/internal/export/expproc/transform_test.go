package expproc

import (
	"testing"

	"github.com/rusq/fsadapter"
)

func Test_mmtransform(t *testing.T) {
	// TODO: automate.
	// MANUAL
	const base = "../../../../../"
	const srcdir = base + "tmp/exportv3"
	const fsaDir = base + "tmp/exportv3/out"
	type args struct {
		fsa    fsadapter.FS
		srcdir string
		id     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				fsa:    fsadapter.NewDirectory(fsaDir),
				srcdir: srcdir,
				id:     "D01MN4X7UGP",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mmtransform(tt.args.fsa, tt.args.srcdir, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("mmtransform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
