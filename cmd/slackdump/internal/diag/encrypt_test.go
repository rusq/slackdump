// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
		defer in.Close()
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
		defer in.Close()
		defer out.Close()
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
