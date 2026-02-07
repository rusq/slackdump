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

import tea "github.com/charmbracelet/bubbletea"

// BoolModel is a model for updating a boolean value.  On startup, it sends
// a message to set the value to the opposite of the current value, and sends
// OnClose message. It has no view.
type BoolModel struct {
	Value *bool
}

func NewBool(ptrBool *bool) BoolModel {
	return BoolModel{Value: ptrBool}
}

func (m BoolModel) Init() tea.Cmd {
	// we have only one goal - to invert the value for the given boolean
	// pointer, when this component activates.
	return cmdSetValue("", !*m.Value)
}

// cmdSetValue returns a command that sets a value to v, key is implementation
// specific, may not be used by the caller.
func cmdSetValue[T any](key string, v T) func() tea.Msg {
	return func() tea.Msg {
		return wmSetValue[T]{key: key, v: v}
	}
}

// wmSetValue is a message that bears a value to set.
type wmSetValue[T any] struct {
	key string
	v   T
}

func (m BoolModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case wmSetValue[bool]:
		*m.Value = msg.v
		return m, OnClose
	}
	return m, nil
}

func (m BoolModel) View() string {
	// View is not being used, but it's here for tests.
	if *m.Value {
		return "[x]"
	}
	return "[ ]"
}
