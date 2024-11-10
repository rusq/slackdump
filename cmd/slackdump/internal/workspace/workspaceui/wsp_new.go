package workspaceui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/menu"
)

type wizModel struct {
	m *menu.Model
}

func WorkspaceNew(ctx context.Context, _ *base.Command, _ []string) error {
	items := []menu.Item{
		{
			ID:   "ezlogin",
			Name: "Login in Browser",
			Help: "Login to Slack in your browser",
		},
		{
			ID:   "token",
			Name: "Token/Cookie",
			Help: "Enter token and cookie that you grabbed from the browser",
		},
		{
			ID:   "secrets",
			Name: "From file with secrets",
			Help: "Read from secrets.txt or .env file",
		},
		{
			Separator: true,
		},
		{
			ID:   "exit",
			Name: "Exit",
			Help: "Exit to main menu",
		},
	}

	m := menu.New("New Workspace", items, true)

	if _, err := tea.NewProgram(&wizModel{m: m}, tea.WithContext(ctx)).Run(); err != nil {
		return err
	}
	return nil
}

func (m *wizModel) Init() tea.Cmd {
	return m.m.Init()
}

func (m *wizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.m.Update(msg)
}

func (m *wizModel) View() string {
	return m.m.View()
}
