package bootstrap

import (
	"context"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
)

// CurrentProviderCtx returns the context with the current provider.
func CurrentProviderCtx(ctx context.Context) (context.Context, error) {
	prov, err := workspace.AuthCurrent(ctx, cfg.CacheDir(), cfg.Workspace, cfg.LegacyBrowser)
	if err != nil {
		return ctx, err
	}
	return auth.WithContext(ctx, prov), nil
}
