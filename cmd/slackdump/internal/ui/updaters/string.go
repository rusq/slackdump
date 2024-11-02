package updaters

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui"
)

type StringModel struct {
	Value     *string
	m         textinput.Model
	errStyle  lipgloss.Style
	finishing bool
}

// NewString creates a new stringUpdateModel
func NewString(ptrStr *string, placeholder string, showPrompt bool, validateFn func(s string) error) StringModel {
	m := textinput.New()
	m.Focus()
	m.Validate = validateFn
	m.EchoMode = textinput.EchoNormal
	m.CharLimit = 255
	m.SetValue(*ptrStr)
	m.Cursor.Style = ui.DefaultTheme().Focused.Cursor
	m.PromptStyle = ui.DefaultTheme().Focused.Title
	m.TextStyle = ui.DefaultTheme().Focused.Text
	m.Placeholder = placeholder
	m.Width = 40
	if !showPrompt {
		m.Prompt = ""
	}
	return StringModel{
		Value:    ptrStr,
		m:        m,
		errStyle: ui.DefaultTheme().Error,
	}
}

func (m StringModel) Init() tea.Cmd {
	return m.m.Focus()
}

func (m StringModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, task := trace.NewTask(context.Background(), "updaters.StringModel.Update")
	defer task.End()

	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.finishing = true
			return m, OnClose
		case "enter":
			if m.m.Err != nil { // if there is an error, don't allow to finish
				return m, nil
			}
			m.finishing = true
			*m.Value = m.m.Value()
			return m, OnClose
		}
	}

	m.m, cmd = m.m.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m StringModel) View() string {
	_, task := trace.NewTask(context.Background(), "updaters.StringModel.View")
	defer task.End()
	if m.finishing {
		return ""
	}
	var buf strings.Builder
	strs := make([]string, 0, 2)
	strs = append(strs, m.m.View())
	if m.m.Err != nil {
		strs = append(strs, "\n"+m.errStyle.Render(m.m.Err.Error()))
	}
	buf.WriteString(lipgloss.JoinVertical(lipgloss.Top, strs...))
	return buf.String()
}

func (m StringModel) Err() error {
	return m.m.Err
}
