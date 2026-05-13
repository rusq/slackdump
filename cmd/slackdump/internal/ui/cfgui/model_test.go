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

package cfgui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
)

func TestModelUpdateNumberShortcutSelectsValidRow(t *testing.T) {
	m := newTestModel(testConfigWithParams(3))
	m.cursor = 0

	updateTestModel(t, m, keyRune('2'))

	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}
}

func TestModelUpdateNumberShortcutIgnoresOutOfRangeRow(t *testing.T) {
	m := newTestModel(testConfigWithParams(3))
	m.cursor = 1

	updateTestModel(t, m, keyRune('9'))

	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want unchanged cursor 1", m.cursor)
	}
}

func TestModelUpdateEnterOnBoolTogglesWithoutChild(t *testing.T) {
	enabled := false
	m := newTestModel(func() Configuration {
		return Configuration{
			{
				Name: "Flags",
				Params: []Parameter{
					{
						Name:    "Enabled",
						Value:   Checkbox(enabled),
						Updater: updaters.NewBool(&enabled),
					},
				},
			},
		}
	})

	_, cmd := updateTestModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if !enabled {
		t.Fatal("enabled = false, want true")
	}
	if m.state != selecting {
		t.Fatalf("state = %v, want selecting", m.state)
	}
	if m.child != nil {
		t.Fatalf("child = %T, want nil", m.child)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want refresh command")
	}
}

func TestModelUpdateEnterOnInlineUpdaterMountsChild(t *testing.T) {
	value := "old"
	m := newTestModel(func() Configuration {
		return Configuration{
			{
				Name: "Strings",
				Params: []Parameter{
					{
						Name:    "Name",
						Value:   value,
						Inline:  true,
						Updater: updaters.NewString(&value, "", false, nil),
					},
				},
			},
		}
	})

	updateTestModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != inline {
		t.Fatalf("state = %v, want inline", m.state)
	}
	if m.child == nil {
		t.Fatal("child = nil, want mounted updater")
	}
}

func TestModelUpdateEnterOnModalUpdaterMountsChild(t *testing.T) {
	value := "old"
	m := newTestModel(func() Configuration {
		return Configuration{
			{
				Name: "Strings",
				Params: []Parameter{
					{
						Name:    "Name",
						Value:   value,
						Updater: updaters.NewString(&value, "", false, nil),
					},
				},
			},
		}
	})

	updateTestModel(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != editing {
		t.Fatalf("state = %v, want editing", m.state)
	}
	if m.child == nil {
		t.Fatal("child = nil, want mounted updater")
	}
}

func newTestModel(cfgFn func() Configuration) *Model {
	m := NewConfigUI(DefaultStyle(), cfgFn)
	m.SetFocus(true)
	return m
}

func updateTestModel(t *testing.T, m *Model, msg tea.Msg) (*Model, tea.Cmd) {
	t.Helper()

	model, cmd := m.Update(msg)
	updated, ok := model.(*Model)
	if !ok {
		t.Fatalf("model = %T, want *Model", model)
	}
	return updated, cmd
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func testConfigWithParams(n int) func() Configuration {
	return func() Configuration {
		params := make([]Parameter, n)
		for i := range params {
			params[i] = Parameter{Name: "Param"}
		}
		return Configuration{
			{
				Name:   "Group",
				Params: params,
			},
		}
	}
}
