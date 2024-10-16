package cfgui

import tea "github.com/charmbracelet/bubbletea"

type boolUpdateModel struct {
	v *bool
}

func (m boolUpdateModel) Init() tea.Cmd {
	return nil
}

func (m boolUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	*m.v = !*m.v
	return m, cmdClose
}

func (m boolUpdateModel) View() string {
	return ""
}
