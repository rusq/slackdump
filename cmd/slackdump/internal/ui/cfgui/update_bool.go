package cfgui

import tea "github.com/charmbracelet/bubbletea"

type boolUpdateModel struct {
	v *bool
}

func (m boolUpdateModel) Init() tea.Cmd {
	// we have only one goal - to invert the value for the given boolean
	// pointer, when this component activates.
	return cmdSetValue("", !*m.v)
}

// cmdSetValue returns a command that sets a value to v, key is implementation
// specific, may not be used by the caller.
func cmdSetValue[T any](key string, v T) func() tea.Msg {
	return func() tea.Msg {
		return wmSetValue[T]{key: key, v: v}
	}
}

// wmSetValue is a message that bears a value to set.
type wmSetValue[T any] struct {
	key string
	v   T
}

func (m boolUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case wmSetValue[bool]:
		*m.v = msg.v
		return m, cmdClose
	}
	return m, nil
}

func (m boolUpdateModel) View() string {
	// View is not being used, but it's here for tests.
	return checkbox(*m.v)
}
