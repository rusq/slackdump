package auth

import (
	"context"
	"fmt"
	"io"
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

type BrowserAuthUIExt interface {
	BrowserAuthUI
	RequestEmail(w io.Writer) (string, error)
	RequestPassword(w io.Writer, account string) (string, error)
	RequestLoginType(w io.Writer) (int, error)
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
		if r.opts.workspace == "" {
			return r, fmt.Errorf("workspace cannot be empty")
		}
	}
	if wsp, err := auth_ui.Sanitize(r.opts.workspace); err != nil {
		return r, err
	} else {
		r.opts.workspace = wsp
	}
	resp, err := r.opts.ui.RequestLoginType(os.Stdout)
	if err != nil {
		return r, err
	}
	var sp simpleProvider
	if resp == auth_ui.LoginSSO {
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
		if username == "" {
			return r, fmt.Errorf("email cannot be empty")
		}
		password, err := r.opts.ui.RequestPassword(os.Stdout, username)
		if err != nil {
			return r, err
		}
		if password == "" {
			return r, fmt.Errorf("password cannot be empty")
		}
		fmt.Fprintln(os.Stderr, "Please wait while Slackdump logs into Slack...")
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
