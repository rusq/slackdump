package updaters

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/rbubbles/filemgr"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
)

type FileModel struct {
	fp       filemgr.Model
	v        *string
	validate func(s string) error
	err      error
	errStyle lipgloss.Style
}

func NewExistingFile(ptrStr *string, f filemgr.Model, validateFn func(s string) error) FileModel {
	f.Focus()
	f.ShowHelp = true
	f.Style = filemgr.Style{
		Normal:    cfg.Theme.Focused.File,
		Directory: cfg.Theme.Focused.Directory,
		Inverted: lipgloss.NewStyle().
			Foreground(cfg.Theme.Focused.FocusedButton.GetForeground()).
			Background(cfg.Theme.Focused.FocusedButton.GetBackground()),
	}
	return FileModel{
		fp:       f,
		v:        ptrStr,
		validate: validateFn,
		errStyle: cfg.Theme.Focused.ErrorMessage,
	}
}

func (m FileModel) Init() tea.Cmd {
	return m.fp.Init()
}

func (m FileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c", "q":
			return m, OnClose
		}
	case filemgr.WMSelected:
		*m.v = msg.Filepath
		if m.validate != nil {
			if err := m.validate(*m.v); err != nil {
				// set error message
				m.err = err
			} else {
				// we are done.
				return m, OnClose
			}
		}
	}

	m.fp, cmd = m.fp.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m FileModel) View() string {
	var buf strings.Builder
	buf.WriteString(m.fp.View())
	if m.err != nil {
		buf.WriteString(m.errStyle.Render(m.err.Error()))
	}
	return buf.String()
}
