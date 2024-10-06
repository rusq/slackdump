package auth

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rusq/slackauth"

	"github.com/rusq/slackdump/v3/auth/auth_ui"
	"github.com/rusq/slackdump/v3/logger"
)

// RODHeadlessTimeout is the default timeout for the headless login flow.
// It is a net time of headless browser interaction, without the browser
// starting time.
const RODHeadlessTimeout = 40 * time.Second

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

type rodOpts struct {
	ui          browserAuthUIExt
	autoTimeout time.Duration
	userAgent   string
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
				ui:          &auth_ui.Huh{},
				autoTimeout: RODHeadlessTimeout,
				userAgent:   "", // slackauth default user agent.
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

	cl, err := slackauth.New(
		r.opts.workspace,
		slackauth.WithChallengeFunc(r.opts.ui.ConfirmationCode),
		slackauth.WithUserAgent(r.opts.userAgent),
		slackauth.WithAutologinTimeout(r.opts.autoTimeout),
	)
	if err != nil {
		return r, err
	}
	defer cl.Close()

	lg := logger.FromContext(ctx)
	var sp simpleProvider
	switch resp {
	case auth_ui.LInteractive:
		lg.Printf("ℹ️ Initialising browser, once the browser appears, login as usual")
		var err error
		sp.Token, sp.Cookie, err = cl.Manual(ctx)
		if err != nil {
			return r, err
		}
	case auth_ui.LHeadless:
		sp, err = headlessFlow(ctx, cl, r.opts.workspace, r.opts.ui)
		if err != nil {
			return r, err
		}
	case auth_ui.LCancel:
		return r, ErrCancelled
	}

	lg.Println("✅ authenticated.")

	return RodAuth{
		simpleProvider: sp,
	}, nil
}

func headlessFlow(ctx context.Context, cl *slackauth.Client, workspace string, ui browserAuthUIExt) (sp simpleProvider, err error) {
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
	logger.FromContext(ctx).Println("⏳ Logging in to Slack, depending on your connection speed, it will take 25-40 seconds...")

	var loginErr error
	sp.Token, sp.Cookie, loginErr = cl.Headless(ctx, username, password)
	if loginErr != nil {
		return sp, loginErr
	}
	return
}
