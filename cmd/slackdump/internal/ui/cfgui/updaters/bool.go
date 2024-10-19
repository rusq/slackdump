package updaters

import tea "github.com/charmbracelet/bubbletea"

type BoolModel struct {
	Value *bool
}

func NewBool(ptrBool *bool) BoolModel {
	return BoolModel{Value: ptrBool}
}

func (m BoolModel) Init() tea.Cmd {
	// we have only one goal - to invert the value for the given boolean
	// pointer, when this component activates.
	return cmdSetValue("", !*m.Value)
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

func (m BoolModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case wmSetValue[bool]:
		*m.Value = msg.v
		return m, OnClose
	}
	return m, nil
}

func (m BoolModel) View() string {
	// View is not being used, but it's here for tests.
	if *m.Value {
		return "[x]"
	}
	return "[ ]"
}
