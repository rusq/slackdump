package updaters

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	datepicker "github.com/ethanefung/bubble-datepicker"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/btime"
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
}

func NewDTTM(ptrTime *time.Time) DateModel {
	m := datepicker.New(*ptrTime)
	t := btime.New(m.Time)
	m.SelectDate()
	focusStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(cfg.Theme.Focused.Title.GetForeground())
	blurStyle := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(cfg.Theme.Blurred.Description.GetForeground())
	return DateModel{
		Value:       ptrTime,
		dm:          m,
		tm:          t,
		focusstyle:  focusStyle,
		blurstyle:   blurStyle,
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

	help := cfg.Theme.Help.Ellipsis.Render("arrow keys: adjust • tab/shift+tab: switch fields • enter: select")

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
			help,
		))
	} else {
		b.WriteString(lipgloss.JoinVertical(
			lipgloss.Center,
			dateStyle.Render(m.dm.View()),
			help,
		))
	}
	return b.String()
}
