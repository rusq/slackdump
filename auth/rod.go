package auth

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/rusq/slackauth"
	"github.com/rusq/slackdump/v2/auth/auth_ui"
)

// RodAuth is an authentication provider that uses a headless or interactive
// browser to authenticate with Slack, depending on the user's choice.  It uses
// rod library to drive the browser via the CDP protocol.
//
// User can choose between:
//   - Email/password login - will be done headlessly
//   - SSO authentication - will open the browser and let the user do the thing.
//   - Cancel - will cancel the login flow.
//
// Headless login is a bit fragile.  If it fails, user should be advised to
// login interactively by choosing SSO auth type.
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
	// RequestLoginType should request the login type from the user and return
	// one of the [auth_ui.LoginType] constants.  The implementation should
	// provide a way to cancel the login flow, returning [auth_ui.LoginCancel].
	RequestLoginType(w io.Writer) (auth_ui.LoginType, error)
	// RequestCreds should request the user's email and password and return
	// them.
	RequestCreds(w io.Writer, workspace string) (email string, passwd string, err error)
	// ConfirmationCode should request the confirmation code from the user and
	// return it.
	ConfirmationCode(email string) (code int, err error)
}

// NewRODAuth constructs new RodAuth provider.
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
	case auth_ui.LInteractive:
		var err error
		sp.Token, sp.Cookie, err = slackauth.Browser(ctx, r.opts.workspace)
		if err != nil {
			return r, err
		}
	case auth_ui.LHeadless:
		sp, err = headlessFlow(ctx, r.opts.workspace, r.opts.ui)
		if err != nil {
			return r, err
		}
	case auth_ui.LCancel:
		return r, ErrCancelled
	}

	fmt.Fprintln(os.Stderr, "authenticated.")

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
	return
}
