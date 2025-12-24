package updaters

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
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
