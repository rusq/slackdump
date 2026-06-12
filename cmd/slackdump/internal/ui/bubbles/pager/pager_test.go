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

package pager

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// content generates n numbered lines.
func content(n int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

func sized(t *testing.T, m *Model, w, h int) *Model {
	t.Helper()
	mod, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	res, ok := mod.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", mod)
	}
	return res
}

func TestModel_View(t *testing.T) {
	t.Run("does not panic before first WindowSizeMsg", func(t *testing.T) {
		m := New("Title", content(100))
		if v := m.View(); v == "" {
			t.Error("View() = empty, want placeholder output")
		}
	})
	t.Run("shows title and first lines after sizing", func(t *testing.T) {
		m := sized(t, New("My Help", content(100)), 80, 24)
		v := m.View()
		if !strings.Contains(v, "My Help") {
			t.Errorf("View() missing title, got:\n%s", v)
		}
		if !strings.Contains(v, "line 0") {
			t.Errorf("View() missing first content line, got:\n%s", v)
		}
		if strings.Contains(v, "line 50") {
			t.Errorf("View() shows line beyond viewport height, got:\n%s", v)
		}
	})
	t.Run("empty when finishing", func(t *testing.T) {
		m := sized(t, New("Title", content(100)), 80, 24)
		mod, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if v := mod.View(); v != "" {
			t.Errorf("View() after quit = %q, want empty", v)
		}
	})
}

func TestModel_Update_quit(t *testing.T) {
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEsc},
		{Type: tea.KeyCtrlC},
	} {
		t.Run(k.String(), func(t *testing.T) {
			m := sized(t, New("Title", content(100)), 80, 24)
			_, cmd := m.Update(k)
			if cmd == nil {
				t.Fatal("Update returned nil cmd, want tea.Quit")
			}
			if msg := cmd(); msg != tea.Quit() {
				t.Errorf("cmd() = %v, want tea.QuitMsg", msg)
			}
		})
	}
}

func TestModel_Update_scroll(t *testing.T) {
	m := sized(t, New("Title", content(100)), 80, 24)
	if !strings.Contains(m.View(), "line 0") {
		t.Fatal("expected viewport at top")
	}
	mod, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = mod.(*Model)
	if strings.Contains(m.View(), "line 0\n") {
		t.Error("viewport did not scroll down on pgdown")
	}
	if m.vp.YOffset == 0 {
		t.Errorf("YOffset = 0 after pgdown, want > 0")
	}
}
