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
// Package datepicker provides a bubble tea component for viewing and selecting
// a date from a monthly view.
package datepicker

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Focus is a value passed to `model.SetFocus` to indicate what component
// controls should be available.
type Focus int

const (
	// FocusNone is a value passed to `model.SetFocus` to ignore all date altering key msgs
	FocusNone Focus = iota
	// FocusHeaderMonth is a value passed to `model.SetFocus` to accept key msgs that change the month
	FocusHeaderMonth
	// FocusHeaderYear is a value passed to `model.SetFocus` to accept key msgs that change the year
	FocusHeaderYear
	// FocusCalendar is a value passed to `model.SetFocus` to accept key msgs that change the week or date
	FocusCalendar
)

//go:generate stringer -type=Focus

// KeyMap is the key bindings for different actions within the datepicker.
type KeyMap struct {
	Up        key.Binding
	Right     key.Binding
	Down      key.Binding
	Left      key.Binding
	FocusPrev key.Binding
	FocusNext key.Binding
	Quit      key.Binding
}

// DefaultKeyMap returns a KeyMap struct with default values
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k")),
		Right:     key.NewBinding(key.WithKeys("right", "l")),
		Down:      key.NewBinding(key.WithKeys("down", "j")),
		Left:      key.NewBinding(key.WithKeys("left", "h")),
		FocusPrev: key.NewBinding(key.WithKeys("shift+tab")),
		FocusNext: key.NewBinding(key.WithKeys("tab")),
		Quit:      key.NewBinding(key.WithKeys("ctrl+c", "q")),
	}
}

// Styles is a struct of lipgloss styles to apply to various elements of the datepicker
type Styles struct {
	HeaderPad lipgloss.Style
	DatePad   lipgloss.Style

	HeaderText   lipgloss.Style
	Text         lipgloss.Style
	SelectedText lipgloss.Style
	FocusedText  lipgloss.Style
}

// DefaultStyles returns a default `Styles` struct
func DefaultStyles() Styles {
	// TODO: refactor for adaptive colors
	r := lipgloss.DefaultRenderer()
	return Styles{
		HeaderPad:    r.NewStyle().Padding(1, 0, 0),
		DatePad:      r.NewStyle().Padding(0, 1, 1),
		HeaderText:   r.NewStyle().Bold(true),
		Text:         r.NewStyle().Foreground(lipgloss.Color("247")),
		SelectedText: r.NewStyle().Bold(true),
		FocusedText:  r.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
	}
}

// Model is a struct that contains the state of the datepicker component and satisfies
// the `tea.Model` interface
type Model struct {
	// Time is the `time.Time` struct that represents the selected date month and year
	Time time.Time

	// KeyMap encodes the keybindings recognized by the model
	KeyMap KeyMap

	// Styles represent the Styles struct used to render the datepicker
	Styles Styles

	// Focused indicates the component which the end user is focused on
	Focused Focus

	// Selected indicates whether a date is Selected in the datepicker
	Selected bool
}

// New returns the Model of the datepicker
func New(time time.Time) Model {
	return Model{
		Time:   time,
		KeyMap: DefaultKeyMap(),
		Styles: DefaultStyles(),

		Focused:  FocusCalendar,
		Selected: false,
	}
}

// Init satisfies the `tea.Model` interface. This sends a nil cmd
func (m Model) Init() tea.Cmd {
	return nil
}

// Update changes the state of the datepicker. Update satisfies the `tea.Model` interface
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.KeyMap.Up):
			m.updateUp()

		case key.Matches(msg, m.KeyMap.Right):
			m.updateRight()

		case key.Matches(msg, m.KeyMap.Down):
			m.updateDown()

		case key.Matches(msg, m.KeyMap.Left):
			m.updateLeft()

		case key.Matches(msg, m.KeyMap.FocusPrev):
			switch m.Focused {
			case FocusHeaderYear:
				m.SetFocus(FocusHeaderMonth)
			case FocusCalendar:
				m.SetFocus(FocusHeaderYear)
			}

		case key.Matches(msg, m.KeyMap.FocusNext):
			switch m.Focused {
			case FocusHeaderMonth:
				m.SetFocus(FocusHeaderYear)
			case FocusHeaderYear:
				m.SetFocus(FocusCalendar)
			}
		}
	}
	return m, nil
}

func (m *Model) updateUp() {
	switch m.Focused {
	case FocusHeaderYear:
		m.LastYear()
	case FocusHeaderMonth:
		m.LastMonth()
	case FocusCalendar:
		m.LastWeek()
	case FocusNone:
		// do nothing
	}
}

func (m *Model) updateRight() {
	switch m.Focused {
	case FocusHeaderYear:
		// do nothing
	case FocusHeaderMonth:
		m.SetFocus(FocusHeaderYear)
	case FocusCalendar:
		m.Tomorrow()
	case FocusNone:
		// do nothing
	}

}
func (m *Model) updateDown() {
	switch m.Focused {
	case FocusHeaderYear:
		m.NextYear()
	case FocusHeaderMonth:
		m.NextMonth()
	case FocusCalendar:
		m.NextWeek()
	case FocusNone:
		// do nothing
	}
}
func (m *Model) updateLeft() {
	switch m.Focused {
	case FocusHeaderYear:
		m.SetFocus(FocusHeaderMonth)
	case FocusHeaderMonth:
		// do nothing
	case FocusCalendar:
		m.Yesterday()
	case FocusNone:
		// do nothing
	}
}

// View renders a month view as a multiline string in the bubbletea application.
// View satisfies the `tea.Model` interface.
func (m Model) View() string {

	var b strings.Builder
	month := m.Time.Month()
	year := m.Time.Year()

	tMonth, tYear := month.String(), strconv.Itoa(year)

	if m.Focused == FocusHeaderMonth {
		tMonth = m.Styles.FocusedText.Render(tMonth)
	} else {
		tMonth = m.Styles.HeaderText.Render(tMonth)
	}

	if m.Focused == FocusHeaderYear {
		tYear = m.Styles.FocusedText.Render(tYear)
	} else {
		tYear = m.Styles.HeaderText.Render(tYear)
	}

	title := m.Styles.HeaderPad.Render(fmt.Sprintf("%s %s\n", tMonth, tYear))

	// get all the dates of the current month
	firstDayOfTheMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)

	lastSundayOfLastMonth := firstDayOfTheMonth.AddDate(0, 0, -1)
	for lastSundayOfLastMonth.Weekday() != time.Sunday {
		lastSundayOfLastMonth = lastSundayOfLastMonth.AddDate(0, 0, -1)
	}

	lastDayOfTheMonth := firstDayOfTheMonth.AddDate(0, 1, -1)

	firstSundayOfNextMonth := lastDayOfTheMonth.AddDate(0, 0, 1)
	for firstSundayOfNextMonth.Weekday() != time.Sunday {
		firstSundayOfNextMonth = firstSundayOfNextMonth.AddDate(0, 0, 1)
	}

	day := lastSundayOfLastMonth
	if firstDayOfTheMonth.Weekday() == time.Sunday {
		day = firstDayOfTheMonth
	}

	weekHeaders := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	for i, h := range weekHeaders {
		weekHeaders[i] = m.Styles.DatePad.Inherit(m.Styles.HeaderText).Render(h)
	}

	cal := [][]string{weekHeaders}
	j := 1

	for day.Before(firstSundayOfNextMonth) {
		if j >= len(cal) {
			cal = append(cal, []string{})
		}
		out := "  "
		if day.Month() == month {
			out = fmt.Sprintf("%02d", day.Day())
		}

		padding := m.Styles.DatePad
		textStyle := m.Styles.Text
		if !m.Selected {
			// skip modifications to the date
		} else if day.Day() == m.Time.Day() && day.Month() == m.Time.Month() && m.Focused == FocusCalendar {
			textStyle = m.Styles.FocusedText
		} else if day.Day() == m.Time.Day() && day.Month() == m.Time.Month() {
			textStyle = m.Styles.SelectedText
		}

		out = padding.Inherit(textStyle).Render(out)
		cal[j] = append(cal[j], out)

		if day.Weekday() == time.Saturday {
			j++
		}
		day = day.AddDate(0, 0, 1)
	}

	rows := []string{title}
	for _, row := range cal {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, row...))
	}
	b.WriteString(lipgloss.JoinVertical(lipgloss.Center, rows...))

	return b.String()
}

// SetsFocus focuses one of the datepicker components. This can also be used to blur
// the datepicker by passing the Focus `FocusNone`.
func (m *Model) SetFocus(f Focus) {
	m.Focused = f
}

// Blur sets the datepicker focus to `FocusNone`
func (m *Model) Blur() {
	m.Focused = FocusNone
}

// SetTime sets the model's `Time` struct and is used as reference to the selected date
func (m *Model) SetTime(t time.Time) {
	m.Time = t
}

// LastWeek sets the model's `Time` struct back 7 days
func (m *Model) LastWeek() {
	m.Time = m.Time.AddDate(0, 0, -7)
}

// NextWeek sets the model's `Time` struct forward 7 days
func (m *Model) NextWeek() {
	m.Time = m.Time.AddDate(0, 0, 7)
}

// Yesterday sets the model's `Time` struct back 1 day
func (m *Model) Yesterday() {
	m.Time = m.Time.AddDate(0, 0, -1)
}

// Tomorrow sets the model's `Time` struct forward 1 day
func (m *Model) Tomorrow() {
	m.Time = m.Time.AddDate(0, 0, 1)
}

// LastMonth sets the model's `Time` struct back 1 month
func (m *Model) LastMonth() {
	m.Time = m.Time.AddDate(0, -1, 0)
}

// NextMonth sets the model's `Time` struct forward 1 month
func (m *Model) NextMonth() {
	m.Time = m.Time.AddDate(0, 1, 0)
}

// LastYear sets the model's `Time` struct back 1 year
func (m *Model) LastYear() {
	m.Time = m.Time.AddDate(-1, 0, 0)
}

// NextYear sets the model's `Time` struct forward 1 year
func (m *Model) NextYear() {
	m.Time = m.Time.AddDate(1, 0, 0)
}

// SelectDate changes the model's Selected to true
func (m *Model) SelectDate() {
	m.Selected = true
}

// UnselectDate changes the model's Selected to false
func (m *Model) UnselectDate() {
	m.Selected = false
}
