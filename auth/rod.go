package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/rusq/slackauth"
	"github.com/rusq/slackdump/v2/auth/auth_ui"
)

type RodAuth struct {
	simpleProvider
	opts rodOpts
}

func (p RodAuth) Type() Type {
	return TypeRod
}

type rodOpts struct {
	ui        BrowserAuthUIExt
	workspace string
}

func RodWithWorkspace(name string) Option {
	return func(o *options) {
		o.rodOpts.workspace = name
	}
}

func NewRODAuth(ctx context.Context, opts ...Option) (RodAuth, error) {
	r := RodAuth{
		opts: rodOpts{
			ui: &auth_ui.CLI{},
		},
	}
	for _, opt := range opts {
		opt(&options{
			rodOpts: &r.opts,
		})
	}
	if r.opts.workspace == "" {
		var err error
		r.opts.workspace, err = r.opts.ui.RequestWorkspace(os.Stdout)
		if err != nil {
			return r, err
		}
	}
	if wsp, err := sanitize(r.opts.workspace); err != nil {
		return r, err
	} else {
		r.opts.workspace = wsp
	}
	yes, err := r.opts.ui.YesNo(os.Stdout, "Does your Slack Workspace use Single Sign On (i.e. Google, Okta, Auth0 etc.)?\n\tIf unsure, say yes")
	if err != nil {
		return r, err
	}
	var sp simpleProvider
	if yes {
		var err error
		sp.Token, sp.Cookie, err = slackauth.Browser(ctx, r.opts.workspace)
		if err != nil {
			return r, err
		}
	} else {
		var err error
		username, err := r.opts.ui.RequestEmail(os.Stdout)
		if err != nil {
			return r, err
		}
		password, err := r.opts.ui.RequestPassword(os.Stdout)
		if err != nil {
			return r, err
		}
		fmt.Fprintln(os.Stderr, "Please wait while slackdump authenticates with Slack...")
		sp.Token, sp.Cookie, err = slackauth.Headless(ctx, r.opts.workspace, username, password)
		if err != nil {
			return r, err
		}
		fmt.Fprintln(os.Stderr, "authenticated.")
	}

	return RodAuth{
		simpleProvider: sp,
	}, nil
}
