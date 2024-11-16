package workspaceui

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type SelectModel struct {
	Selected string

	table    table.Model
	finished bool
	style    style
}

func NewSelectModel(columns []table.Column, rows []table.Row) SelectModel {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.Styles{
		Header:   ui.DefaultTheme().Focused.Title.Padding(0, 1),
		Selected: ui.DefaultTheme().Focused.SelectedLine.Bold(true),
		Cell:     ui.DefaultTheme().Focused.Text.Padding(0, 1),
	}
	t.SetStyles(s)
	t.Focus()
	return SelectModel{
		table: t,
		style: style{
			FocusedBorder: ui.DefaultTheme().Focused.Border,
		},
	}
}

type style struct {
	FocusedBorder lipgloss.Style
}

func (m SelectModel) Init() tea.Cmd { return nil }

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.finished = true
			cmds = append(cmds, tea.Quit)
		case "enter":
			m.Selected = m.table.SelectedRow()[1]
			m.finished = true
			cmds = append(cmds, tea.Quit)
		}
	}
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m SelectModel) View() string {
	if m.finished {
		return "" // don't render the table if we've selected a workspace
	}
	return m.style.FocusedBorder.Render((m.table.View()) + "\n\n" + ui.HuhTheme().Help.Ellipsis.Render("Select the workspace with arrow keys, press [Enter] to confirm, [Esc] to cancel."))
}
