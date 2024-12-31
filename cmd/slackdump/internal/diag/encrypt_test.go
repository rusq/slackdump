package diag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseArgs(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		in, out, arm, err := parseArgs([]string{})
		require.NoError(t, err)
		assert.Equal(t, os.Stdin, in)
		assert.Equal(t, os.Stdout, out)
		assert.True(t, arm)
	})
	t.Run("one arg", func(t *testing.T) {
		in, out, arm, err := parseArgs([]string{"-"})
		require.NoError(t, err)
		assert.Equal(t, os.Stdin, in)
		assert.Equal(t, os.Stdout, out)
		assert.True(t, arm)
	})
	t.Run("two args", func(t *testing.T) {
		in, out, arm, err := parseArgs([]string{"-", "-"})
		require.NoError(t, err)
		assert.Equal(t, os.Stdin, in)
		assert.Equal(t, os.Stdout, out)
		assert.True(t, arm)
	})
	t.Run("two args, first is file", func(t *testing.T) {
		dir := t.TempDir()
		f, _ := os.Create(filepath.Join(dir, "foo"))
		f.Close()

		in, out, arm, err := parseArgs([]string{filepath.Join(dir, "foo"), "-"})
		require.NoError(t, err)
		defer out.Close()
		assert.Equal(t, os.Stdout, out)
		assert.True(t, arm)

		if inf, ok := in.(*os.File); ok {
			assert.Equal(t, filepath.Join(dir, "foo"), inf.Name())
		} else {
			t.Errorf("input is not a file")
		}
	})
	t.Run("two args, second is file", func(t *testing.T) {
		dir := t.TempDir()
		in, out, arm, err := parseArgs([]string{"-", filepath.Join(dir, "bar")})
		require.NoError(t, err)
		defer out.Close()
		assert.Equal(t, os.Stdin, in)
		assert.False(t, arm)

		if outf, ok := out.(*os.File); ok {
			assert.Equal(t, filepath.Join(dir, "bar"), outf.Name())
		} else {
			t.Errorf("output is not a file")
		}
	})
	t.Run("two args, both are files", func(t *testing.T) {
		dir := t.TempDir()
		f, _ := os.Create(filepath.Join(dir, "foo"))
		f.Close()

		in, out, arm, err := parseArgs([]string{filepath.Join(dir, "foo"), filepath.Join(dir, "bar")})
		require.NoError(t, err)
		assert.False(t, arm)

		if inf, ok := in.(*os.File); ok {
			assert.Equal(t, filepath.Join(dir, "foo"), inf.Name())
		} else {
			t.Errorf("input is not a file")
		}

		if outf, ok := out.(*os.File); ok {
			assert.Equal(t, filepath.Join(dir, "bar"), outf.Name())
		} else {
			t.Errorf("output is not a file")
		}
	})
}
