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

package ui

import (
	"slices"
	"testing"
)

func TestDefaultHuhKeymap(t *testing.T) {
	a, b := DefaultHuhKeymap(), DefaultHuhKeymap()
	if a == b {
		t.Fatal("DefaultHuhKeymap() returned the same instance twice; huh forms mutate keymap state, so callers must not share one")
	}
	// the repository overrides must be applied to every instance.
	if got := a.Quit.Help().Key; got != "esc/ctrl+c" {
		t.Errorf("Quit help key = %q, want %q", got, "esc/ctrl+c")
	}
	if got := a.Select.Up.Help().Key; got != KeyUp {
		t.Errorf("Select.Up help key = %q, want %q", got, KeyUp)
	}
	for _, k := range []string{"up", "k", "ctrl+k", "ctrl+p"} {
		if !slices.Contains(a.Select.Up.Keys(), k) {
			t.Errorf("Select.Up missing key %q", k)
		}
	}
}
