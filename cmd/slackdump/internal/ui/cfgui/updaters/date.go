package updaters

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	datepicker "github.com/ethanefung/bubble-datepicker"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/btime"
)

type DateModel struct {
	Value       *time.Time
	dm          datepicker.Model
	tm          *btime.Model
	finishing   bool
	timeEnabled bool
	state       state
}

func NewDTTM(ptrTime *time.Time) DateModel {
	m := datepicker.New(*ptrTime)
	t := btime.New(m.Time)
	m.SelectDate()
	return DateModel{
		Value:       ptrTime,
		dm:          m,
		tm:          t,
		timeEnabled: true,
	}
}

func (m DateModel) Init() tea.Cmd {
	return m.dm.Init()
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
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, OnClose
		case "enter":
			d := m.dm.Time
			t := m.tm.Value()
			*m.Value = time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, m.Value.Location())
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
	if m.timeEnabled {
		b.WriteString(lipgloss.JoinVertical(lipgloss.Center, m.dm.View(), m.tm.View()))
	} else {
		b.WriteString(m.dm.View())
	}
	b.WriteString("\n\n" + m.dm.Styles.Text.Render("Use arrow keys to navigate, tab/shift+tab to switch between fields, and enter to select."))
	return b.String()
}
