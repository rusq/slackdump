// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
