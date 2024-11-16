package workspaceui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	keymap   selKeymap
	help     help.Model
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
		keymap: defSelKeymap(),
		help:   help.New(),
	}
}

type style struct {
	FocusedBorder lipgloss.Style
}

type selKeymap struct {
	Select key.Binding
	Delete key.Binding
	Quit   key.Binding
}

func (k selKeymap) Bindings() []key.Binding {
	return []key.Binding{k.Select, k.Delete, k.Quit}
}

func defSelKeymap() selKeymap {
	return selKeymap{
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")),
		Delete: key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("del", "Delete")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("esc", "Quit")),
	}
}

func (m SelectModel) Init() tea.Cmd { return nil }

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Quit):
			m.finished = true
			cmds = append(cmds, tea.Quit)
		case key.Matches(msg, m.keymap.Select):
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
	return m.style.FocusedBorder.Render((m.table.View()) + "\n\n" + m.help.ShortHelpView(m.keymap.Bindings()))
}
