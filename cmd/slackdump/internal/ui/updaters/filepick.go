package updaters

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/filemgr"
)

type FilepickModel struct {
	fp          filemgr.Model
	v           *string
	validate    func(s string) error
	err         error
	borderStyle lipgloss.Style
	errStyle    lipgloss.Style
}

func NewFilepickModel(ptrStr *string, f filemgr.Model, validateFn func(s string) error) FilepickModel {
	f.Focus()
	f.ShowHelp = true
	f.Style = filemgr.Style{
		Normal:    ui.DefaultTheme().Focused.UnselectedFile,
		Directory: ui.DefaultTheme().Focused.Directory,
		Inverted:  ui.DefaultTheme().Focused.SelectedFile,
		CurDir:    ui.DefaultTheme().Focused.Description,
	}
	return FilepickModel{
		fp:          f,
		v:           ptrStr,
		validate:    validateFn,
		borderStyle: ui.DefaultTheme().Focused.Border,
		errStyle:    ui.DefaultTheme().Error,
	}
}

func (m FilepickModel) Init() tea.Cmd {
	return m.fp.Init()
}

func (m FilepickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m FilepickModel) View() string {
	var buf strings.Builder
	buf.WriteString(m.fp.View())
	if m.err != nil {
		buf.WriteString(m.errStyle.Render(m.err.Error()))
	}
	return m.borderStyle.Render(buf.String())
}
