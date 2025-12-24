package updaters

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type PicklistModel[T comparable] struct {
	s         huh.Field
	help      help.Model
	finishing bool

	// value
	initial T
	ptr     *T
}

func NewPicklist[T comparable](v *T, s *huh.Select[T]) *PicklistModel[T] {
	m := &PicklistModel[T]{
		s: s.Value(v).
			Description("Select an option").
			WithTheme(ui.HuhTheme()).
			WithKeyMap(huh.NewDefaultKeyMap()),
		help: help.New(),

		initial: *v,
		ptr:     v,
	}
	return m
}

func (m *PicklistModel[T]) Init() tea.Cmd {
	return tea.Batch(m.s.Init(), m.s.Focus())
}

func (m *PicklistModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		mod, cmd := m.s.Update(msg)
		if mod, ok := mod.(huh.Field); ok {
			m.s = mod
		}
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *PicklistModel[T]) View() string {
	if m.finishing {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.s.View(), m.help.ShortHelpView(m.s.KeyBinds()))
}
