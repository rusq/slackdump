package transform

import (
	"context"
	"path/filepath"
	"runtime/trace"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/state"
)

const whereTheTempIsAt = "../../tmp"

func TestStandard_Transform(t *testing.T) {
	t.Skip()
	ctx, task := trace.NewTask(context.Background(), "TestStandard_Transform")
	defer task.End()
	// MANUAL
	fs := fsadapter.NewDirectory(filepath.Join(whereTheTempIsAt, "manual"))
	s := NewStandard(fs)
	st, err := state.Load(filepath.Join(whereTheTempIsAt, "C01SPFM1KNY.state"))
	if err != nil {
		t.Fatalf("state.Load(): %s", err)
	}
	if err := s.Transform(ctx, whereTheTempIsAt, st); err != nil {
		t.Fatal(err)
	}
}

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
			if err := stdConvert(fsa, cd, name); err != nil {
				t.Fatalf("failed on i=%d, name=%s: %s", i, name, err)
			}
		}
	})
}
