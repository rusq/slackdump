package transform

import (
	"context"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
)

func Test_transform(t *testing.T) {
	// TODO: automate.
	// MANUAL
	const (
		base   = "../../../"
		srcdir = base + "tmp/exportv3"
		fsaDir = base + "tmp/exportv3/out"
	)
	type args struct {
		ctx    context.Context
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
				ctx:    context.Background(),
				fsa:    fsadapter.NewDirectory(fsaDir),
				srcdir: srcdir,
				// id:     "D01MN4X7UGP",
				id: "C01SPFM1KNY",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd, err := chunk.OpenDir(tt.args.srcdir)
			if err != nil {
				t.Fatal(err)
			}
			if err := transform(tt.args.ctx, tt.args.fsa, cd, chunk.FileID(tt.args.id), nil, nil); (err != nil) != tt.wantErr {
				t.Errorf("transform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
