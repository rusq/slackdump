package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
	"github.com/rusq/slackdump/v3/internal/cache"
)

func CurrentOrNewProviderCtx(ctx context.Context) (context.Context, error) {
	prov, err := workspace.AuthCurrent(ctx, cfg.CacheDir(), cfg.Workspace, cfg.LegacyBrowser)
	if err != nil {
		if errors.Is(err, cache.ErrNoWorkspaces) {
			// ask to create a new workspace
			if err := workspace.CmdWspNew.Run(ctx, workspace.CmdWspNew, []string{}); err != nil {
				return ctx, fmt.Errorf("auth error: %w", err)
			}
			// one more time...
			prov, err = workspace.AuthCurrent(ctx, cfg.CacheDir(), cfg.Workspace, cfg.LegacyBrowser)
			if err != nil {
				return ctx, err
			}
		} else {
			return ctx, err
		}
	}
	return auth.WithContext(ctx, prov), nil
}
