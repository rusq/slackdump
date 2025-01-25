package archive

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDirectory(t *testing.T) {
	t.Run("creates a directory", func(t *testing.T) {
		tmpdir := t.TempDir()
		cd, err := NewDirectory(tmpdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cd == nil {
			t.Fatal("expected a directory, got nil")
		}
		defer cd.Close()
		assert.Equal(t, tmpdir, cd.Name())
	})
}
