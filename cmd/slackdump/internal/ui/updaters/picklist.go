package updaters

import (
	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type Model[T comparable] struct {
	s         huh.Field
	help      help.Model
	finishing bool

	// value
	initial T
	ptr     *T
}

func NewPicklist[T comparable](v *T, s *huh.Select[T]) *Model[T] {
	m := &Model[T]{
		s: s.Value(v).
			Description("Select an option").
			WithTheme(ui.HuhTheme).
			WithKeyMap(huh.NewDefaultKeyMap()),
		help: help.New(),

		initial: *v,
		ptr:     v,
	}
	return m
}

func (m *Model[T]) Init() tea.Cmd {
	return tea.Batch(m.s.Init(), m.s.Focus())
}

func (m *Model[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *Model[T]) View() string {
	if m.finishing {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.s.View(), m.help.ShortHelpView(m.s.KeyBinds()))
}
