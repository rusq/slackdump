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

var DefaultHuhKeymap = huh.NewDefaultKeyMap()

func init() {
	// redefinition of some of the default keys.
	DefaultHuhKeymap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"), key.WithHelp("esc/ctrl+c", "quit"))
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

func KeySelectBinding(help string) key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp(KeyEnter, help))
}

func KeyQuitBinding() key.Binding {
	return key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp(KeyQuitAll, "quit"))
}

func NewHelp() help.Model {
	h := help.New()
	h.Styles = DefaultTheme().Help
	return h
}
