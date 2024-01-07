package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

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
	switch resp {
	case auth_ui.LoginSSO:
		var err error
		sp.Token, sp.Cookie, err = slackauth.Browser(ctx, r.opts.workspace)
		if err != nil {
			return r, err
		}
	case auth_ui.LoginEmail:
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
		fmt.Println("Logging in to Slack, depending on your connection speed, it usually takes 10-20 seconds...")

		var loginErr error
		sp.Token, sp.Cookie, loginErr = slackauth.Headless(
			ctx,
			r.opts.workspace,
			username,
			password,
			slackauth.WithChallengeFunc(r.opts.ui.ConfirmationCode),
		)
		if loginErr != nil {
			return r, loginErr
		}

		fmt.Fprintln(os.Stderr, "authenticated.")
	case auth_ui.LoginCancel:
		return r, ErrCancelled
	}

	return RodAuth{
		simpleProvider: sp,
	}, nil
}
