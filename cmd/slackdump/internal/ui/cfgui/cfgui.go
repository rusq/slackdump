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
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/updaters"
)

// programWrap wraps the UI model to implement the tea.Model interface with
// tea.Quit message emitted when the user presses ESC or Ctrl+C.
type programWrap struct {
	m         *Model
	finishing bool
}

func newProgramWrap(m *Model) tea.Model {
	return programWrap{m: m}
}

func (m programWrap) Init() tea.Cmd {
	return m.m.Init()
}

func (m programWrap) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case updaters.WMClose:
		if msg.WndID == ModelID {
			m.finishing = true
			cmds = append(cmds, tea.Quit)
		}
	}

	mod, cmd := m.m.Update(msg)
	if mod, ok := mod.(*Model); ok {
		m.m = mod
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m programWrap) View() string {
	if m.finishing {
		return ""
	}
	return m.m.View()
}

// Global initialises and runs the configuration UI.
func Global(ctx context.Context) error {
	m := NewConfigUI(DefaultStyle(), globalConfig)
	m.SetFocus(true)
	p := tea.NewProgram(newProgramWrap(m))
	_, err := p.Run()
	return err
}

func GlobalConfig() Configuration {
	return globalConfig()
}

func Local(ctx context.Context, cfgFn func() Configuration) error {
	p := tea.NewProgram(newProgramWrap(NewConfigUI(DefaultStyle(), cfgFn)))
	_, err := p.Run()
	return err
}
