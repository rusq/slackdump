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
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
)

type multiselectModel struct {
	m    huh.Field
	help help.Model

	finishing bool
	initial   []string
	ptr       *[]string
}

func NewMultiSelect(v *[]string, m *huh.MultiSelect[string]) *multiselectModel {
	msm := multiselectModel{
		m:       m.Value(v).WithKeyMap(huh.NewDefaultKeyMap()).WithTheme(ui.HuhTheme()),
		help:    help.New(),
		ptr:     v,
		initial: make([]string, len(*v)),
	}
	copy(msm.initial, *v)
	return &msm
}

func (m *multiselectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			// restore initial value
			*m.ptr = m.initial
			m.finishing = true
			cmds = append(cmds, OnClose)
		case "enter":
			m.finishing = true
			cmds = append(cmds, OnClose)
		}
	}
	{
		// update the select control
		mod, cmd := m.m.Update(msg)
		if mod, ok := mod.(huh.Field); ok {
			m.m = mod
		}
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *multiselectModel) Init() tea.Cmd {
	return tea.Batch(m.m.Init(), m.m.Focus())
}

func (m *multiselectModel) View() string {
	if m.finishing {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.m.View(), m.help.ShortHelpView(m.m.KeyBinds()))

}
