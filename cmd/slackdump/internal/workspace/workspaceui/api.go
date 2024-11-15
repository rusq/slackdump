package workspaceui

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/auth/auth_ui"
)

//go:generate mockgen -package workspaceui -destination=test_mock_manager.go -source api.go manager
type manager interface {
	SaveProvider(workspace string, p auth.Provider) error
	Select(workspace string) error
}

// createAndSelect creates a new workspace with the given provider and selects it.
// It returns the workspace name on success.
func createAndSelect(ctx context.Context, m manager, prov auth.Provider) (string, error) {
	authInfo, err := prov.Test(ctx)
	if err != nil {
		return "", err
	}

	wsp, err := auth_ui.Sanitize(authInfo.URL)
	if err != nil {
		return "", err
	}
	if wsp == "" {
		return "", errors.New("workspace name is empty")
	}
	if err := m.SaveProvider(wsp, prov); err != nil {
		return "", err
	}
	if err := m.Select(wsp); err != nil {
		return "", err
	}
	return wsp, nil
}
