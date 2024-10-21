package updaters

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	datepicker "github.com/ethanefung/bubble-datepicker"
)

type DateModel struct {
	Value       *time.Time
	dm          datepicker.Model
	finishing   bool
	timeEnabled bool
}

func NewDTTM(ptrTime *time.Time) DateModel {
	m := datepicker.New(*ptrTime)
	m.SelectDate()
	return DateModel{
		Value:       ptrTime,
		dm:          m,
		timeEnabled: true,
	}
}

func (m DateModel) Init() tea.Cmd {
	return m.dm.Init()
}

func (m DateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, OnClose
		case "enter":
			*m.Value = m.dm.Time
			m.finishing = true
			return m, OnClose
		}
	}

	m.dm, cmd = m.dm.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m DateModel) View() string {
	var b strings.Builder
	b.WriteString(m.dm.View())
	if m.timeEnabled {
		b.WriteString("\n\nTime:  " + m.Value.Format("15:04:05") + " (UTC)")
	}
	b.WriteString("\n\n" + m.dm.Styles.Text.Render("Use arrow keys to navigate, tab/shift+tab to switch between fields, and enter to select."))
	return b.String()
}
