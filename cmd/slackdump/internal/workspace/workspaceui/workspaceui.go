package workspaceui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/menu"
	"github.com/rusq/slackdump/v3/internal/cache"
)

func WorkspaceNew(ctx context.Context, _ *base.Command, _ []string) error {
	const (
		actLogin     = "ezlogin"
		actToken     = "token"
		actTokenFile = "tokenfile"
		actSecrets   = "secrets"
		actExit      = "exit"
	)

	mgr, err := cache.NewManager(cfg.CacheDir())
	if err != nil {
		return err
	}

	items := []menu.Item{
		{
			ID:   actLogin,
			Name: "Login in Browser",
			Help: "Login to Slack in your browser",
		},
		{
			ID:   actToken,
			Name: "Token/Cookie",
			Help: "Enter token and cookie that you grabbed from the browser",
		},
		{
			ID:   actTokenFile,
			Name: "Token/Cookie from file",
			Help: "Provide token value and cookies from file",
		},
		{
			ID:   actSecrets,
			Name: "From file with secrets",
			Help: "Read from secrets.txt or .env file",
		},
		{
			Separator: true,
		},
		{
			ID:   actExit,
			Name: "Exit",
			Help: "Exit to main menu",
		},
	}

LOOP:
	for {
		m := menu.New("New Workspace", items, true)
		if _, err := tea.NewProgram(&wizModel{m: m}, tea.WithContext(ctx)).Run(); err != nil {
			return err
		}
		if m.Cancelled {
			break LOOP
		}
		var err error
		switch m.Selected.ID {
		case actToken:
			err = prgTokenCookie(ctx, mgr)
		case actTokenFile:
			err = prgTokenCookieFile(ctx, mgr)
		case actExit:
			break LOOP
		}
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				continue
			}
			return err
		}
	}

	return nil
}

// wizModel is a wrapper around the menu.
type wizModel struct{ m *menu.Model }

func (m *wizModel) Init() tea.Cmd                           { return m.m.Init() }
func (m *wizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m.m.Update(msg) }
func (m *wizModel) View() string                            { return m.m.View() }
