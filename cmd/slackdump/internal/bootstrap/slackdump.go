package bootstrap

import (
	"context"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/workspace"
)

// SlackdumpSession returns the Slackdump Session initialised with the provider
// from context and a standard set of options initialised from the
// configuration.  One can provide additional options to override the
// defaults.
func SlackdumpSession(ctx context.Context, opts ...slackdump.Option) (*slackdump.Session, error) {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	var stdOpts = []slackdump.Option{
		slackdump.WithLogger(cfg.Log),
		slackdump.WithForceEnterprise(cfg.ForceEnterprise),
		slackdump.WithLimits(cfg.Limits),
	}

	stdOpts = append(stdOpts, opts...)
	return slackdump.New(
		ctx,
		prov,
		stdOpts...,
	)
}

// CurrentProviderCtx returns the context with the current provider.
func CurrentProviderCtx(ctx context.Context) (context.Context, error) {
	prov, err := workspace.AuthCurrent(ctx, cfg.CacheDir(), cfg.Workspace, cfg.LegacyBrowser)
	if err != nil {
		return ctx, err
	}
	return auth.WithContext(ctx, prov), nil
}
