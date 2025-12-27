package bootstrap

import (
	"context"
	"fmt"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/client"
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

	stdOpts := []slackdump.Option{
		slackdump.WithLogger(cfg.Log),
		slackdump.WithForceEnterprise(cfg.ForceEnterprise),
		slackdump.WithLimits(cfg.Limits),
	}

	stdOpts = append(stdOpts, opts...)
	return slackdump.NewNoValidate(
		ctx,
		prov,
		stdOpts...,
	)
}

// Slack returns the Slack client initialised with the provider from context
// and a standard set of options initialised from the configuration.
func Slack(ctx context.Context, opts ...client.Option) (client.Slack, error) {
	prov, err := auth.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w", err)
	}
	opts = append(opts, client.WithEnterprise(cfg.ForceEnterprise))
	client, err := client.New(ctx, prov, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating new client: %w", err)
	}
	return client, nil
}
