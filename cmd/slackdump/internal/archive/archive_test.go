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
