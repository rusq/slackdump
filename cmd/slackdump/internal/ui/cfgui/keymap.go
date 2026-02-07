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

import "github.com/charmbracelet/bubbles/key"

type Keymap struct {
	Up      key.Binding
	Down    key.Binding
	Home    key.Binding
	End     key.Binding
	Refresh key.Binding
	Select  key.Binding
	Quit    key.Binding
}

func DefaultKeymap() *Keymap {
	return &Keymap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
		Home:    key.NewBinding(key.WithKeys("home"), key.WithHelp("home/end", "top/bottom")),
		End:     key.NewBinding(key.WithKeys("end")),
		Refresh: key.NewBinding(key.WithKeys("f5", "ctrl+r"), key.WithHelp("f5", "refresh")),
		Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
		Quit:    key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k *Keymap) Bindings() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Home, k.Refresh, k.Select, k.Quit}
}
