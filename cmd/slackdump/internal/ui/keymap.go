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
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

const (
	KeyEnter    = "enter"
	KeyEsc      = "esc"
	KeyQuit     = "q"
	KeyQuitAll  = "q/esc/ctrl+c"
	KeyCtrlC    = "ctrl+c"
	KeyCtrlR    = "ctrl+r"
	KeyUp       = "↑/k"
	KeyDown     = "↓/j"
	KeyUpDown   = "↑/↓"
	KeyArrows   = "←↑↓→"
	KeyTab      = "tab"
	KeyShiftTab = "shift+tab"
	KeyHomeEnd  = "home/end"
	KeyRefresh  = "f5/ctrl+r"
	KeyBack     = "backspace"
	KeyDelete   = "delete"
)

// DefaultHuhKeymap returns a fresh huh keymap with the repository
// help-label overrides applied.  It returns a new instance on every call:
// huh forms mutate binding enabled-state during navigation, so live forms
// must not share a keymap.
func DefaultHuhKeymap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("esc/ctrl+c", "quit"))
	km.Select.Up = key.NewBinding(key.WithKeys("up", "k", "ctrl+k", "ctrl+p"), key.WithHelp(KeyUp, "up"))
	km.Select.Down = key.NewBinding(key.WithKeys("down", "j", "ctrl+j", "ctrl+n"), key.WithHelp(KeyDown, "down"))
	km.MultiSelect.Up = key.NewBinding(key.WithKeys("up", "k", "ctrl+p"), key.WithHelp(KeyUp, "up"))
	km.MultiSelect.Down = key.NewBinding(key.WithKeys("down", "j", "ctrl+n"), key.WithHelp(KeyDown, "down"))
	km.FilePicker.Up = key.NewBinding(key.WithKeys("up", "k", "ctrl+k", "ctrl+p"), key.WithHelp(KeyUp, "up"), key.WithDisabled())
	km.FilePicker.Down = key.NewBinding(key.WithKeys("down", "j", "ctrl+j", "ctrl+n"), key.WithHelp(KeyDown, "down"), key.WithDisabled())
	return km
}

func KeyUpBinding() key.Binding {
	return key.NewBinding(key.WithKeys("up", "k"), key.WithHelp(KeyUp, "up"))
}

func KeyDownBinding() key.Binding {
	return key.NewBinding(key.WithKeys("down", "j"), key.WithHelp(KeyDown, "down"))
}

func KeyHomeBinding() key.Binding {
	return key.NewBinding(key.WithKeys("home"), key.WithHelp(KeyHomeEnd, "top/bottom"))
}

func KeyEndBinding() key.Binding {
	return key.NewBinding(key.WithKeys("end"))
}

func KeyRefreshBinding() key.Binding {
	return key.NewBinding(key.WithKeys("f5", "ctrl+r"), key.WithHelp(KeyRefresh, "refresh"))
}

func KeySelectBinding(desc string) key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp(KeyEnter, desc))
}

func KeyQuitBinding() key.Binding {
	return key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp(KeyQuitAll, "quit"))
}

func NewHelp() help.Model {
	h := help.New()
	h.Styles = DefaultTheme().Help
	return h
}
