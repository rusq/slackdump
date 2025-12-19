package updaters

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sosodev/duration"
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

type ISODurationModel struct {
	Value *duration.Duration
	sv    string

	m StringModel
}

func NewISODuration(value *duration.Duration, showPrompt bool) ISODurationModel {
	dm := ISODurationModel{
		Value: value,
		sv:    value.String(),
	}
	dm.m = NewString(&dm.sv, "p1wt1h20m55s", showPrompt, ValidateISODuration)
	return dm
}

func (m ISODurationModel) Init() tea.Cmd {
	return m.m.Init()
}

func (m ISODurationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		d, _ := duration.Parse(strings.ToUpper(*m.m.Value))
		*m.Value = *d
	}

	return m, tea.Batch(cmds...)
}

func (m ISODurationModel) View() string {
	return m.m.View()
}

func ValidateISODuration(s string) error {
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "P") {
		s = "P" + s
	}
	_, err := duration.Parse(s)
	return err
}
