package transform

import (
	"testing"
)

func TestExport_RestoreState(t *testing.T) {
	t.Run("manual", func(t *testing.T) {
		t.Skip()
		_, err := ExportState("../../../tmp/kiwi1.zip")
		if err != nil {
			t.Fatal(err)
		}
		t.Error("x")
	})
}
