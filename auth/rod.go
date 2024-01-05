package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/huh/spinner"
	"github.com/rusq/slackauth"
	"github.com/rusq/slackdump/v2/auth/auth_ui"
)

type RodAuth struct {
	simpleProvider
	opts options
}

func (p RodAuth) Type() Type {
	return TypeRod
}

type rodOpts struct {
	ui browserAuthUIExt
}

type browserAuthUIExt interface {
	BrowserAuthUI
	RequestLoginType(w io.Writer) (int, error)
	RequestCreds(w io.Writer, workspace string) (email string, passwd string, err error)
}

const expectedLoginDuration = 16 * time.Second

func NewRODAuth(ctx context.Context, opts ...Option) (RodAuth, error) {
	r := RodAuth{
		opts: options{
			rodOpts: rodOpts{
				ui: &auth_ui.Huh{},
			},
		},
	}
	for _, opt := range opts {
		opt(&r.opts)
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
		username, password, err := r.opts.ui.RequestCreds(os.Stdout, r.opts.workspace)
		if err != nil {
			return r, err
		}
		if username == "" {
			return r, fmt.Errorf("email cannot be empty")
		}
		if password == "" {
			return r, fmt.Errorf("password cannot be empty")
		}
		sctx, cancel := context.WithCancel(ctx)
		go spinner.New().Title("Logging in...").Context(sctx).Run()
		sp.Token, sp.Cookie, err = slackauth.Headless(ctx, r.opts.workspace, username, password)
		cancel()
		if err != nil {
			return r, err
		}
		fmt.Fprintln(os.Stderr, "authenticated.")
	}

	return RodAuth{
		simpleProvider: sp,
	}, nil
}
