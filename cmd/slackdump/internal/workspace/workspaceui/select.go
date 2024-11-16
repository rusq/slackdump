package workspaceui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type SelectModel struct {
	Selected  string
	m         manager
	refreshFn TableRefreshFunc

	table    table.Model
	finished bool
	style    style
	keymap   selKeymap
	help     help.Model
	lastErr  error
}

type TableRefreshFunc func() ([]table.Column, []table.Row, error)

func NewSelectModel(m manager, refreshFn TableRefreshFunc) SelectModel {
	columns, rows, err := refreshFn()
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
		table:     t,
		m:         m,
		refreshFn: refreshFn,
		style: style{
			FocusedBorder: ui.DefaultTheme().Focused.Border,
			Title:         ui.DefaultTheme().Focused.Title,
			Description:   ui.DefaultTheme().Focused.Description,
			Error:         ui.DefaultTheme().Error,
		},
		keymap:  defSelKeymap(),
		help:    help.New(),
		lastErr: err,
	}
}

type style struct {
	FocusedBorder lipgloss.Style
	Title         lipgloss.Style
	Description   lipgloss.Style
	Error         lipgloss.Style
}

type selKeymap struct {
	Select  key.Binding
	Delete  key.Binding
	Quit    key.Binding
	Refresh key.Binding
}

func (k selKeymap) Bindings() []key.Binding {
	return []key.Binding{k.Select, k.Delete, k.Quit}
}

func defSelKeymap() selKeymap {
	return selKeymap{
		Select:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Select")),
		Delete:  key.NewBinding(key.WithKeys("x", "delete"), key.WithHelp("del", "Delete")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("esc", "Quit")),
		Refresh: key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("^r", "Refresh")),
	}
}

func (m SelectModel) Init() tea.Cmd { return nil }

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var refresh = func() {
		columns, rows, err := m.refreshFn()
		m.table.SetColumns(columns)
		m.table.SetRows(rows)
		m.lastErr = err
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Select):
			if len(m.table.SelectedRow()) == 0 {
				break
			}
			m.Selected = m.table.SelectedRow()[1]
			fallthrough
		case key.Matches(msg, m.keymap.Quit):
			m.finished = true
			cmds = append(cmds, tea.Quit)
		case key.Matches(msg, m.keymap.Delete):
			if len(m.table.SelectedRow()) == 0 {
				break
			}
			m.lastErr = m.m.Delete(m.table.SelectedRow()[1])
			refresh()
		case key.Matches(msg, m.keymap.Refresh):
			refresh()
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
	var b strings.Builder

	b.WriteString(m.style.Title.Render("Slackdump Workspaces") + "\n")
	b.WriteString(m.style.Description.Render("Select a workspace to work with") + "\n\n")
	b.WriteString(m.table.View() + "\n")
	if m.lastErr != nil {
		b.WriteString(m.style.Error.Render(m.lastErr.Error()) + "\n")
	}
	b.WriteString(m.help.ShortHelpView(m.keymap.Bindings()))

	return m.style.FocusedBorder.Render(b.String())
}
