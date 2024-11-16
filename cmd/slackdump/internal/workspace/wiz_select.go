package workspace

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/workspaceui"
	"github.com/rusq/slackdump/v3/internal/cache"
	"github.com/rusq/slackdump/v3/logger"
)

// TODO: organise as a self-sufficient model with proper error handling.

func wizSelect(ctx context.Context, cmd *base.Command, args []string) error {
	m, err := cache.NewManager(cfg.CacheDir())
	if err != nil {
		base.SetExitStatus(base.SCacheError)
		return err
	}

	sm, err := newWspSelectModel(ctx, m)
	if err != nil {
		return err
	}

	mod, err := tea.NewProgram(sm).Run()
	if err != nil {
		return fmt.Errorf("workspace select wizard error: %w", err)
	}
	if newWsp := mod.(workspaceui.SelectModel).Selected; newWsp != "" {
		if err := m.Select(newWsp); err != nil {
			base.SetExitStatus(base.SWorkspaceError)
			return fmt.Errorf("error setting the current workspace: %s", err)
		}
		logger.FromContext(ctx).Debugf("selected workspace: %s", newWsp)
	}

	return nil
}

// newWspSelectModel creates a new workspace selection model.
func newWspSelectModel(ctx context.Context, m manager) (tea.Model, error) {
	wspList, err := m.List()
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			if err := workspaceui.ShowUI(ctx, true); err != nil {
				return nil, err
			}
			// retry
			wspList, err = m.List()
			if err != nil {
				return nil, err
			}
		} else {
			base.SetExitStatus(base.SUserError)
			return nil, err
		}
	}

	current, err := m.Current()
	if err != nil {
		base.SetExitStatus(base.SWorkspaceError)
		return nil, fmt.Errorf("error getting the current workspace: %s", err)
	}

	columns := []table.Column{
		{Title: "C", Width: 1},
		{Title: "Name", Width: 14},
		{Title: "Team", Width: 15},
		{Title: "User", Width: 15},
		{Title: "Status", Width: 30},
	}

	var rows []table.Row
	for _, w := range wspInfo(ctx, m, current, wspList) {
		rows = append(rows, table.Row{w[0], w[1], w[4], w[5], w[6]})
	}

	return workspaceui.NewSelectModel(columns, rows), nil
}
