package workspaceui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/ui/bubbles/menu"
	"github.com/rusq/slackdump/v3/internal/cache"
)

//go:generate mockgen -package workspaceui -destination=test_mock_manager.go -source api.go manager
type manager interface {
	CreateAndSelect(ctx context.Context, p auth.Provider) (string, error)
}

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
			Help: "Opens the browser and lets you login in a familiar way.",
		},
		{
			ID:   actToken,
			Name: "Token/Cookie",
			Help: "Enter token and cookie that you grabbed from the browser.",
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

	// new workspace methods
	var methods = map[string]func(context.Context, manager) error{
		actLogin:     ezLogin3000,
		actToken:     prgTokenCookie,
		actTokenFile: prgTokenCookieFile,
		actSecrets:   fileWithSecrets,
	}

	var lastID string = actLogin
LOOP:
	for {
		m := menu.New("New Workspace", items, true)
		m.Select(lastID)
		if _, err := tea.NewProgram(&wizModel{m: m}, tea.WithContext(ctx)).Run(); err != nil {
			return err
		}
		lastID = m.Selected.ID
		if m.Cancelled {
			break LOOP
		}
		if m.Selected.ID == actExit {
			break LOOP
		}
		fn, ok := methods[m.Selected.ID]
		if !ok {
			return errors.New("internal error:  unhandled login option")
		}
		if err := fn(ctx, mgr); err != nil {
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
