package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_initTrace(t *testing.T) {
	t.Run("initialises trace file", func(t *testing.T) {
		testTraceFile := filepath.Join(t.TempDir(), "trace.out")
		stop := initTrace(testTraceFile)
		t.Cleanup(stop)
		assert.FileExists(t, testTraceFile)
	})
}
