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

package wizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/workspace"
)

type model struct {
	form     *huh.Form
	val      string
	finished bool
}

const kSelection = "selection" // selection key

func newModel(m *menu) model {
	var options []huh.Option[string]
	for i, name := range m.names {
		var text = fmt.Sprintf("%-10s - %s", name, m.items[i].Description)
		if m.items[i].Description == "" {
			text = fmt.Sprintf("%-10s", name)
		}
		options = append(options, huh.NewOption(text, name))
	}
	return model{
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Key(kSelection).
					Title(m.title).
					Description("Slack workspace:  " + workspace.CurrentName()).
					Options(options...),
			),
		).WithTheme(ui.HuhTheme()),
	}
}

func (m *model) Init() tea.Cmd {
	return m.form.Init()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.finished = true
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted {
		m.val = m.form.GetString(kSelection)
		cmds = append(cmds, tea.Quit)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.finished {
		return ""
	}
	return m.form.View()
}
