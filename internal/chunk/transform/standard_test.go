package transform

import (
	"context"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/nametmpl"
)

func Test_stdConvert(t *testing.T) {
	var testNames = []chunk.FileID{
		"CHYLGDP0D-1682335799.257359",
		"CHYLGDP0D-1682375167.836499",
		"CHM82GF99",
	}
	t.Run("manual", func(t *testing.T) {
		const testDir = "../../../tmp/3"
		cd, err := chunk.OpenDir(testDir)
		if err != nil {
			t.Fatal(err)
		}
		defer cd.Close()
		fsa, err := fsadapter.New("output-dump.zip")
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		cvt := StdConverter{
			cd:   cd,
			fsa:  fsa,
			tmpl: nametmpl.NewDefault(),
		}

		for i, name := range testNames {
			if err := cvt.Convert(context.Background(), name); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}
