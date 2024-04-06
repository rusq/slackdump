package status

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	v      viewport.Model
	params *Parameters
}

func New(height int, style lipgloss.Style, params []Parameter) Model {
	if height < 1 {
		height = len(params)
	}
	var idx = make(map[string]int, len(params))
	for i, p := range params {
		idx[p.Name] = i
	}
	return Model{
		v:      viewport.Model{Height: height, Style: style},
		params: NewParameters(params...),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.v.Width = msg.Width
	}
	return m, nil
}

func (m Model) View() string {
	m.v.SetContent(m.params.String())
	return m.v.View()
}
