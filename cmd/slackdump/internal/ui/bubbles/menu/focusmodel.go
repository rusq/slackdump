package menu

import tea "github.com/charmbracelet/bubbletea"

type FocusModel interface {
	tea.Model
	SetFocus(bool)
	IsFocused() bool
	// Reset should reset the model to its initial state.
	Reset()
}
