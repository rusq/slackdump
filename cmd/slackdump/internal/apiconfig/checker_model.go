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
package apiconfig

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/filemgr"
)

type checkerModel struct {
	files      filemgr.Model
	view       viewport.Model
	BlurStyle  lipgloss.Style
	FocusStyle lipgloss.Style
	state      wizState
	width      int
	finishing  bool
}

func (m checkerModel) Init() tea.Cmd {
	return tea.Batch(m.files.Init(), m.view.Init())
}

type wizState int

const (
	wizStateFile wizState = iota
	wizStateView
)

func (m checkerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var keymsg bool
	switch msg := msg.(type) {
	case wmSetText:
		m.view.Style.Foreground(msg.style.GetForeground())
		m.view.SetContent(msg.text)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.view.Width = msg.Width - filemgr.Width
	case tea.KeyMsg:
		keymsg = true
		switch msg.String() {
		case "ctrl+c", "q":
			m.finishing = true
			return m, tea.Quit
		case "tab":
			switch m.state {
			case wizStateFile:
				m.state = wizStateView
				m.files.Blur()
			case wizStateView:
				m.state = wizStateFile
				m.files.Focus()
			}
		}
	case filemgr.WMSelected:
		filename := msg.Filepath
		if err := CheckFile(filename); err != nil {
			cmds = append(cmds, wcmdErr(filename, err))
		} else {
			cmds = append(cmds, wcmdOK(filename))
		}
	}

	var cmd tea.Cmd
	m.files, cmd = m.files.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if !keymsg || m.state == wizStateView {
		// we do not propagate key messages to the viewport.
		m.view, cmd = m.view.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m checkerModel) View() string {
	const crlf = "\r\n"
	if m.finishing {
		return ""
	}

	styFiles := m.FocusStyle
	styView := m.BlurStyle
	switch m.state {
	case wizStateView:
		styFiles = m.BlurStyle
		styView = m.FocusStyle
	}
	var buf strings.Builder
	buf.WriteString(strings.Repeat("⎯", m.width) + crlf)
	buf.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, styFiles.Render(m.files.View()), styView.Render(m.view.View())) + crlf)
	buf.WriteString(strings.Repeat("⎯", m.width))
	return buf.String()
}

type wmSetText struct {
	text  string
	style lipgloss.Style
}

func wcmdErr(_ string, err error) tea.Cmd {
	return func() tea.Msg {
		return wmSetText{
			text:  err.Error(),
			style: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		}
	}
}

func wcmdOK(filename string) tea.Cmd {
	return func() tea.Msg {
		return wmSetText{
			text:  "Config file OK: " + filename,
			style: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		}
	}
}
