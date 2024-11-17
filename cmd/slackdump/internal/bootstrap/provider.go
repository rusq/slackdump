package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace/workspaceui"
	"github.com/rusq/slackdump/v3/internal/cache"
)

func CurrentOrNewProviderCtx(ctx context.Context) (context.Context, error) {
	cachedir := cfg.CacheDir()
	prov, err := workspace.AuthCurrent(ctx, cachedir, cfg.Workspace, cfg.LegacyBrowser)
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			// ask to create a new workspace
			if err := workspaceui.ShowUI(ctx, workspaceui.WithQuickLogin(), workspaceui.WithTitle("No workspaces, please choose a login method")); err != nil {
				return ctx, fmt.Errorf("auth error: %w", err)
			}
			// one more time...
			prov, err = workspace.AuthCurrent(ctx, cachedir, cfg.Workspace, cfg.LegacyBrowser)
			if err != nil {
				return ctx, err
			}
		} else {
			return ctx, err
		}
	}
	return auth.WithContext(ctx, prov), nil
}
