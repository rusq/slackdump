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

package osext

import (
	"os"
	"testing"
)

func TestRemoveOnClose(t *testing.T) {
	d := t.TempDir()
	t.Run("removes the file on close", func(t *testing.T) {
		f, err := os.CreateTemp(d, "test")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r := RemoveOnClose(f)
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(f.Name()); !os.IsNotExist(err) {
			t.Errorf("file %s still exists", f.Name())
		}
	})
}

func TestRemoveWrapper_Name(t *testing.T) {
	d := t.TempDir()
	t.Run("returns the filename", func(t *testing.T) {
		f, err := os.CreateTemp(d, "test")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		r := RemoveOnClose(f)
		if r.Name() != f.Name() {
			t.Errorf("Name() = %s, want %s", r.Name(), f.Name())
		}
	})
}
