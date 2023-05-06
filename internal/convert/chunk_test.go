package convert

import (
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
)

func TestChunkToExport_Validate(t *testing.T) {
	const (
		testSrcDir = "../../tmp/slackdump_20230506_120330" // TODO: fix manual nature of this/obfuscate
	)
	srcDir, err := chunk.OpenDir(testSrcDir)
	if err != nil {
		t.Fatal(err)
	}
	var testTrgDir = t.TempDir()

	type fields struct {
		Src          *chunk.Directory
		Trg          fsadapter.FS
		UploadDir    string
		IncludeFiles bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"empty", fields{}, true},
		{"no source", fields{Trg: fsadapter.NewDirectory(testTrgDir)}, true},
		{"no target", fields{Src: srcDir}, true},
		{
			"valid",
			fields{
				Src:          srcDir,
				Trg:          fsadapter.NewDirectory(testTrgDir),
				UploadDir:    "__uploads",
				IncludeFiles: true,
			},
			false,
		},
		{
			"upload not exist, but we don't need it",
			fields{
				Src: srcDir, Trg: fsadapter.NewDirectory(testTrgDir),
				UploadDir:    "$$notexist$$",
				IncludeFiles: false,
			},
			false,
		},
		{"upload not exist, and we need it",
			fields{
				Src:          srcDir,
				Trg:          fsadapter.NewDirectory(testTrgDir),
				UploadDir:    "$$notexist$$",
				IncludeFiles: true,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ChunkToExport{
				Src:          tt.fields.Src,
				Trg:          tt.fields.Trg,
				UploadDir:    tt.fields.UploadDir,
				IncludeFiles: tt.fields.IncludeFiles,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ChunkToExport.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
