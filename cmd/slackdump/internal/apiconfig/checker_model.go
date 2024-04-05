package apiconfig

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/filemgr"
)

type checkerModel struct {
	files     filemgr.Model
	view      viewport.Model
	width     int
	finishing bool
}

func (m checkerModel) Init() tea.Cmd {
	return tea.Batch(m.files.Init(), m.view.Init())
}

func (m checkerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case wmSetText:
		m.view.Style.Foreground(msg.style.GetForeground())
		m.view.SetContent(msg.text)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.view.Width = msg.Width - filemgr.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.finishing = true
			return m, tea.Quit
		}
	case filemgr.WMSelected:
		filename := msg.Filepath
		if err := checkFile(filename); err != nil {
			cmds = append(cmds, wcmdErr(filename, err))
		} else {
			cmds = append(cmds, wcmdOK(filename))
		}
	}

	var cmd tea.Cmd
	m.files, cmd = m.files.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.view, cmd = m.view.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m checkerModel) View() string {
	if m.finishing {
		return ""
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, m.files.View(), m.view.View())
}

type wmSetText struct {
	text  string
	style lipgloss.Style
}

func wcmdErr(_ string, err error) tea.Cmd {
	return func() tea.Msg {
		return wmSetText{
			text:  err.Error(),
			style: lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		}
	}
}

func wcmdOK(filename string) tea.Cmd {
	return func() tea.Msg {
		return wmSetText{
			text:  "Config file OK: " + filename,
			style: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		}
	}
}
