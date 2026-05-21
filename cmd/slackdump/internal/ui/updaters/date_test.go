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

package updaters

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/datepicker"
)

func Test_defaultDateKeymap(t *testing.T) {
	km := defaultDateKeymap()

	for _, k := range km.Cancel.Keys() {
		if slices.Contains(km.Arrows.Keys(), k) {
			t.Fatalf("arrows binding contains cancel key %q", k)
		}
	}
	for _, k := range []string{"up", "down", "left", "right", "h", "j", "k", "l"} {
		if !slices.Contains(km.Arrows.Keys(), k) {
			t.Errorf("arrows binding missing movement key %q", k)
		}
	}
	for _, k := range []string{"esc", "q", "ctrl+c"} {
		if !slices.Contains(km.Cancel.Keys(), k) {
			t.Errorf("cancel binding missing key %q", k)
		}
	}

	assertHelp(t, km.Arrows, ui.KeyArrows, "move")
	assertHelp(t, km.Select, ui.KeyEnter, "select")
	assertHelp(t, km.Clear, ui.KeyBack, "clear")
	assertHelp(t, km.Cancel, ui.KeyQuitAll, "cancel")
}

func TestDateModel_Update(t *testing.T) {
	t.Run("enter combines selected date and edited time preserving value location", func(t *testing.T) {
		loc := time.FixedZone("test", 3*60*60)
		value := time.Date(2026, time.May, 1, 1, 2, 3, 0, loc)
		m := NewDTTM(&value)
		m.dm.SetTime(time.Date(2027, time.June, 15, 0, 0, 0, 0, time.UTC))
		m.tm.SetTime(time.Date(2000, time.January, 1, 14, 35, 9, 0, time.UTC))

		got, cmd := m.Update(keyMsg("enter"))
		assertOnClose(t, cmd)
		assertDateModel(t, got)

		want := time.Date(2027, time.June, 15, 14, 35, 9, 0, loc)
		if !value.Equal(want) || value.Location() != loc {
			t.Fatalf("value = %v (%v), want %v (%v)", value, value.Location(), want, loc)
		}
	})

	t.Run("backspace clears value and closes", func(t *testing.T) {
		value := time.Date(2026, time.May, 1, 1, 2, 3, 0, time.UTC)
		m := NewDTTM(&value)

		got, cmd := m.Update(keyMsg("backspace"))
		assertOnClose(t, cmd)
		assertDateModel(t, got)

		if !value.IsZero() {
			t.Fatalf("value = %v, want zero time", value)
		}
	})

	t.Run("cancel keys close without mutating value", func(t *testing.T) {
		for _, k := range []string{"esc", "q", "ctrl+c"} {
			t.Run(k, func(t *testing.T) {
				value := time.Date(2026, time.May, 1, 1, 2, 3, 0, time.UTC)
				want := value
				m := NewDTTM(&value)
				m.dm.SetTime(time.Date(2027, time.June, 15, 0, 0, 0, 0, time.UTC))
				m.tm.SetTime(time.Date(2000, time.January, 1, 14, 35, 9, 0, time.UTC))

				got, cmd := m.Update(keyMsg(k))
				assertOnClose(t, cmd)
				assertDateModel(t, got)

				if !value.Equal(want) || value.Location() != want.Location() {
					t.Fatalf("value = %v, want unchanged %v", value, want)
				}
			})
		}
	})

	t.Run("tab focuses time editor from calendar grid", func(t *testing.T) {
		value := time.Date(2026, time.May, 1, 1, 2, 3, 0, time.UTC)
		m := NewDTTM(&value)
		m.state = scalendar
		m.dm.SetFocus(datepicker.FocusCalendar)

		got, cmd := m.Update(keyMsg("tab"))
		if cmd != nil {
			t.Fatalf("cmd = %v, want nil", cmd)
		}
		dm := assertDateModel(t, got)
		if dm.state != stime {
			t.Fatalf("state = %v, want %v", dm.state, stime)
		}
		if !dm.tm.Focused {
			t.Fatal("time editor is not focused")
		}
	})

	t.Run("shift tab returns focus from time editor to calendar", func(t *testing.T) {
		value := time.Date(2026, time.May, 1, 1, 2, 3, 0, time.UTC)
		m := NewDTTM(&value)
		m.state = stime
		m.tm.Focus()

		got, cmd := m.Update(keyMsg("shift+tab"))
		if cmd != nil {
			t.Fatalf("cmd = %v, want nil", cmd)
		}
		dm := assertDateModel(t, got)
		if dm.state != scalendar {
			t.Fatalf("state = %v, want %v", dm.state, scalendar)
		}
		if dm.tm.Focused {
			t.Fatal("time editor is focused")
		}
	})
}

func assertHelp(t *testing.T, b key.Binding, wantKey, wantDesc string) {
	t.Helper()
	if got := b.Help(); got.Key != wantKey || got.Desc != wantDesc {
		t.Fatalf("help = %q %q, want %q %q", got.Key, got.Desc, wantKey, wantDesc)
	}
}

func assertOnClose(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("cmd = nil, want OnClose")
	}
	if !reflect.DeepEqual(cmd(), OnClose()) {
		t.Fatalf("cmd() = %#v, want %#v", cmd(), OnClose())
	}
}

func assertDateModel(t *testing.T, model tea.Model) DateModel {
	t.Helper()
	dm, ok := model.(DateModel)
	if !ok {
		t.Fatalf("model = %T, want DateModel", model)
	}
	return dm
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}
