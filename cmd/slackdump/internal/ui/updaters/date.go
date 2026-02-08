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
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/btime"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/ui/bubbles/datepicker"
)

type DateModel struct {
	Value       *time.Time
	dm          datepicker.Model
	tm          *btime.Model
	focusstyle  lipgloss.Style
	blurstyle   lipgloss.Style
	finishing   bool
	timeEnabled bool
	state       state
	keymap      dateKeymap
	help        help.Model
}

func NewDTTM(ptrTime *time.Time) DateModel {
	m := datepicker.New(*ptrTime)
	m.Styles = datepicker.Styles{
		HeaderPad:    lipgloss.NewStyle().Padding(1, 0, 0),
		DatePad:      lipgloss.NewStyle().Padding(0, 1, 1),
		HeaderText:   lipgloss.NewStyle().Bold(true),
		Text:         lipgloss.NewStyle().Foreground(lipgloss.Color("247")),
		SelectedText: lipgloss.NewStyle().Bold(true),
		FocusedText:  lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
	}
	t := btime.New(m.Time)
	m.SelectDate()
	return DateModel{
		Value:       ptrTime,
		dm:          m,
		tm:          t,
		focusstyle:  ui.DefaultTheme().Focused.Border,
		blurstyle:   ui.DefaultTheme().Blurred.Border,
		keymap:      defaultDateKeymap(),
		timeEnabled: true,
		help:        help.New(),
	}
}

type dateKeymap struct {
	NextField key.Binding
	PrevField key.Binding
	Arrows    key.Binding
	Select    key.Binding
	Cancel    key.Binding
	Clear     key.Binding
}

func defaultDateKeymap() dateKeymap {
	return dateKeymap{
		NextField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("↹", "next")),
		PrevField: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("⇧ + ↹", "prev")),
		Arrows:    key.NewBinding(key.WithKeys("esc", "ctrl+c", "q"), key.WithHelp("←↑↓→", "move")),
		Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "select")),
		Cancel:    key.NewBinding(key.WithKeys("esc", "ctrl+c", "q"), key.WithHelp("Esc", "cancel")),
		Clear:     key.NewBinding(key.WithKeys("backspace"), key.WithHelp("backspace", "clear")),
	}
}

func (m dateKeymap) keybindings() [][]key.Binding {
	return [][]key.Binding{
		{m.NextField, m.PrevField, m.Arrows},
		{m.Select, m.Cancel, m.Clear},
	}
}

func (m DateModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.Value == nil || m.Value.IsZero() {
		cmds = append(cmds, cmdSetValue("", time.Now()))
	}
	cmds = append(cmds, m.dm.Init(), m.tm.Init())
	return tea.Batch(cmds...)
}

type state int

const (
	scalendar state = iota
	stime
)

func (m DateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case wmSetValue[time.Time]:
		*m.Value = msg.v
		m.dm.SetTime(msg.v)
		m.tm.SetTime(msg.v)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			m.finishing = true
			return m, OnClose
		case "enter":
			d := m.dm.Time
			t := m.tm.Value()
			*m.Value = time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, m.Value.Location())
			m.finishing = true
			return m, OnClose
		case "backspace":
			*m.Value = time.Time{}
			m.finishing = true
			return m, OnClose
		case "tab":
			switch m.state {
			case scalendar:
				if !m.timeEnabled || m.dm.Focused != datepicker.FocusCalendar {
					break
				}
				m.state = stime
				m.tm.Focus()
				return m, nil
			case stime:
				// ignore tab in time mode.
				return m, nil
			}
		case "shift+tab":
			switch m.state {
			case scalendar:
				break
			case stime:
				m.state = scalendar
				m.tm.Blur()
				return m, nil
			}
		}
	}

	switch m.state {
	case scalendar:
		m.dm, cmd = m.dm.Update(msg)
	case stime:
		m.tm, cmd = m.tm.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m DateModel) View() string {
	if m.finishing {
		return ""
	}

	var b strings.Builder

	help := m.help.FullHelpView(m.keymap.keybindings())

	var dateStyle lipgloss.Style
	var timeStyle lipgloss.Style

	if m.state == scalendar {
		dateStyle = m.focusstyle
		timeStyle = m.blurstyle
	} else {
		dateStyle = m.blurstyle
		timeStyle = m.focusstyle
	}

	if m.timeEnabled {
		b.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			dateStyle.Render(m.dm.View()),
			timeStyle.Render(m.tm.View()),
		))
	} else {
		b.WriteString(dateStyle.Render(m.dm.View()))
	}
	return lipgloss.JoinVertical(lipgloss.Left, b.String(), help)
}
