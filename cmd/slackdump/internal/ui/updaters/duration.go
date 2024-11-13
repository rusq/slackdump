package updaters

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// DurationModel is a model for updating a time.Duration value.  It is a wrapper
// around a StringModel.
type DurationModel struct {
	Value *time.Duration
	sv    string // string value

	m StringModel
}

func ValidateDuration(s string) error {
	_, err := time.ParseDuration(s)
	return err
}

func NewDuration(value *time.Duration, showPrompt bool) DurationModel {
	dm := DurationModel{
		Value: value,
		sv:    value.String(),
	}
	dm.m = NewString(&dm.sv, "1h20m55s", showPrompt, ValidateDuration)
	return dm
}

func (m DurationModel) Init() tea.Cmd {
	return m.m.Init()
}

func (m DurationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	{
		mod, cmd := m.m.Update(msg)
		if mod, ok := mod.(StringModel); ok {
			m.m = mod
		}
		cmds = append(cmds, cmd)
	}
	if m.m.finishing {
		// update the value
		d, _ := time.ParseDuration(*m.m.Value)
		*m.Value = d
	}

	return m, tea.Batch(cmds...)
}

func (m DurationModel) View() string {
	return m.m.View()
}
