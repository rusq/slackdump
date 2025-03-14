package transform

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/nametmpl"
	"github.com/rusq/slackdump/v3/internal/source"
)

func Test_stdConvert(t *testing.T) {
	testNames := []chunk.FileID{
		"CHYLGDP0D-1682335799.257359",
		"CHYLGDP0D-1682375167.836499",
		"CHM82GF99",
	}
	t.Run("manual", func(t *testing.T) {
		testDir := filepath.Join("..", "..", "..", "tmp", "3")
		fixtures.SkipIfNotExist(t, testDir)

		ctx := context.Background()

		src, err := source.Load(ctx, testDir)
		if err != nil {
			t.Fatal(err)
		}
		defer src.Close()
		tmp := t.TempDir()
		fsa, err := fsadapter.New(filepath.Join(tmp, "output-dump.zip"))
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		cvt := DumpConverter{
			src:  src,
			fsa:  fsa,
			tmpl: nametmpl.NewDefault(),
		}

		for i, name := range testNames {
			if err := cvt.Convert(ctx, name); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}
