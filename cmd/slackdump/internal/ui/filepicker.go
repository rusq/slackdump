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
package ui

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

type FileSystemModel struct {
	filepicker   filepicker.Model
	SelectedFile string
	quitting     bool
	err          error
	Prompt       string
}

func NewFilePicker(prompt string, homedir string, allowedExt ...string) FileSystemModel {
	fp := filepicker.New()
	fp.AllowedTypes = allowedExt
	fp.CurrentDirectory = homedir
	fp.Styles.Cursor = HuhTheme().Focused.SelectedOption

	return FileSystemModel{
		filepicker: fp,
		Prompt:     prompt,
	}
}

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m FileSystemModel) Init() tea.Cmd {
	return m.filepicker.Init()
}

func (m FileSystemModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker, cmd = m.filepicker.Update(msg)

	// Did the user select a file?
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		// Get the path of the selected file.
		m.SelectedFile = path
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(path + " is not valid.")
		m.SelectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m FileSystemModel) View() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.SelectedFile == "" {
		s.WriteString(m.Prompt)
	} else {
		s.WriteString("Selected file: " + m.filepicker.Styles.Selected.Render(m.SelectedFile))
	}
	s.WriteString("\n\n" + m.filepicker.View() + "\n")
	return s.String()
}
