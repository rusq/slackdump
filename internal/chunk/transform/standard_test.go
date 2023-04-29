package transform

import (
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
		fsa, err := fsadapter.New("output-dump.zip")
		if err != nil {
			t.Fatal(err)
		}
		defer fsa.Close()
		for i, name := range testNames {
			if err := stdConvert(fsa, cd, name, nametmpl.NewDefault()); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}
