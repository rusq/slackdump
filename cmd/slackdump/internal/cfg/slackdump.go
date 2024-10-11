package cfg

import (
	"context"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/auth"
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
		slackdump.WithLogger(Log),
		slackdump.WithForceEnterprise(ForceEnterprise),
		slackdump.WithLimits(Limits),
	}

	stdOpts = append(stdOpts, opts...)
	return slackdump.New(
		ctx,
		prov,
		stdOpts...,
	)
}
