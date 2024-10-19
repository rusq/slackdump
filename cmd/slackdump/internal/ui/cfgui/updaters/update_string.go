package updaters

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StringModel struct {
	Value    *string
	m        textinput.Model
	errStyle lipgloss.Style
}

// NewString creates a new stringUpdateModel
func NewString(ptrStr *string, validateFn func(s string) error) StringModel {
	m := textinput.New()
	m.Focus()
	m.Validate = validateFn
	m.EchoMode = textinput.EchoNormal
	m.CharLimit = 80
	m.SetValue(*ptrStr)

	return StringModel{
		Value:    ptrStr,
		m:        m,
		errStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	}
}

func (m StringModel) Init() tea.Cmd {
	return m.m.Focus()
}

func (m StringModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, OnClose
		case "enter":
			*m.Value = m.m.Value()
			return m, OnClose
		}
	}

	m.m, cmd = m.m.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m StringModel) View() string {
	return m.m.View()
}
