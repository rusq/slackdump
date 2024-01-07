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
	ConfirmationCode(email string) (code int, err error)
}

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
	switch resp {
	case auth_ui.LoginSSO:
		var err error
		sp.Token, sp.Cookie, err = slackauth.Browser(ctx, r.opts.workspace)
		if err != nil {
			return r, err
		}
	case auth_ui.LoginEmail:
		sp, err = headlessFlow(ctx, r.opts.workspace, r.opts.ui)
		if err != nil {
			return r, err
		}
		fmt.Fprintln(os.Stderr, "authenticated.")
	case auth_ui.LoginCancel:
		return r, ErrCancelled
	}

	return RodAuth{
		simpleProvider: sp,
	}, nil
}

func headlessFlow(ctx context.Context, workspace string, ui browserAuthUIExt) (sp simpleProvider, err error) {
	username, password, err := ui.RequestCreds(os.Stdout, workspace)
	if err != nil {
		return sp, err
	}
	if username == "" {
		return sp, fmt.Errorf("email cannot be empty")
	}
	if password == "" {
		return sp, fmt.Errorf("password cannot be empty")
	}
	fmt.Println("Logging in to Slack, depending on your connection speed, it will take 15-30 seconds...")

	var loginErr error
	sp.Token, sp.Cookie, loginErr = slackauth.Headless(
		ctx,
		workspace,
		username,
		password,
		slackauth.WithChallengeFunc(ui.ConfirmationCode),
	)
	if loginErr != nil {
		return sp, loginErr
	}

	fmt.Fprintln(os.Stderr, "authenticated.")
	return
}
