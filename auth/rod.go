// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
package auth

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/rusq/slackauth"

	"github.com/rusq/slackdump/v4/auth/auth_ui"
	"github.com/rusq/slackdump/v4/internal/osext"
	"github.com/rusq/slackdump/v4/internal/structures"
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
	ui             browserAuthUIExt
	autoTimeout    time.Duration
	userAgent      string
	usermode       bool
	bundledBrowser bool
}

func (ro rodOpts) slackauthOpts() []slackauth.Option {
	sopts := []slackauth.Option{
		slackauth.WithChallengeFunc(ro.ui.ConfirmationCode),
		slackauth.WithAutologinTimeout(ro.autoTimeout),
	}
	if ro.userAgent != "" {
		sopts = append(sopts, slackauth.WithUserAgent(ro.userAgent))
	}
	if ro.usermode {
		sopts = append(sopts, slackauth.WithForceUser())
	}
	if ro.bundledBrowser {
		sopts = append(sopts, slackauth.WithBundledBrowser())
	}
	return sopts
}

type browserAuthUIExt interface {
	// RequestLoginType should request the login type from the user and return
	// one of the [auth_ui.LoginType] constants.  The implementation should
	// provide a way to cancel the login flow, returning [auth_ui.LoginCancel].
	RequestLoginType(ctx context.Context, w io.Writer, workspace string) (auth_ui.LoginOpts, error)
	// RequestCreds should request the user's email and password and return
	// them.
	RequestCreds(ctx context.Context, w io.Writer, workspace string) (email string, passwd string, err error)
	// ConfirmationCode should request the confirmation code from the user and
	// return it.  Callback function is called to indicate that the code is
	// requested.
	ConfirmationCode(email string) (code int, err error)
	// RequestQR should request the URL-encoded png image and return it.
	RequestQR(ctx context.Context, w io.Writer) (encImage string, err error)
}

// NewRODAuth constructs new RodAuth provider.
func NewRODAuth(ctx context.Context, opts ...Option) (RodAuth, error) {
	if osext.IsDocker() || !osext.IsInteractive() {
		return RodAuth{}, &Error{Err: ErrNotSupported, Msg: "browser auth is not supported in dumb terminals, use token/cookie auth instead"}
	}
	r := RodAuth{
		opts: options{
			rodOpts: rodOpts{
				ui:             &auth_ui.Huh{},
				autoTimeout:    RODHeadlessTimeout,
				userAgent:      "", // slackauth default user agent.
				usermode:       false,
				bundledBrowser: false,
			},
		},
	}
	for _, opt := range opts {
		opt(&r.opts)
	}
	if wsp, err := structures.ExtractWorkspace(r.opts.workspace); err != nil {
		return r, err
	} else {
		r.opts.workspace = wsp
	}

	resp, err := r.opts.ui.RequestLoginType(ctx, os.Stdout, r.opts.workspace)
	if err != nil {
		return r, err
	}
	sopts := r.opts.slackauthOpts()
	if resp.Type == auth_ui.LUserBrowser {
		// it doesn't need to know that this browser is just a puppet in the
		// masterful hands.
		sopts = append(sopts, slackauth.WithForceUser(), slackauth.WithLocalBrowser(resp.BrowserPath))
	}

	cl, err := slackauth.New(
		resp.Workspace,
		sopts...,
	)
	if err != nil {
		return r, err
	}
	defer cl.Close()

	lg := slog.Default()
	t := time.Now()
	var sp simpleProvider

	switch resp.Type {
	case auth_ui.LInteractive, auth_ui.LUserBrowser:
		lg.InfoContext(ctx, "ℹ️ Initialising browser, once the browser appears, login as usual")
		sp.Token, sp.Cookie, err = cl.Manual(ctx)
	case auth_ui.LHeadless:
		sp, err = headlessFlow(ctx, cl, resp.Workspace, r.opts.ui)
	case auth_ui.LMobileSignin:
		sp, err = qrFlow(ctx, cl, r.opts.ui)
	case auth_ui.LCancel:
		return r, ErrCancelled
	}
	if err != nil {
		return r, err
	}

	lg.InfoContext(ctx, "✅ authenticated", "time_taken", time.Since(t).String())

	return RodAuth{
		simpleProvider: sp,
	}, nil
}

func headlessFlow(ctx context.Context, cl *slackauth.Client, workspace string, ui browserAuthUIExt) (sp simpleProvider, err error) {
	username, password, err := ui.RequestCreds(ctx, os.Stdout, workspace)
	if err != nil {
		return sp, err
	}
	if username == "" {
		return sp, fmt.Errorf("email cannot be empty")
	}
	if password == "" {
		return sp, fmt.Errorf("password cannot be empty")
	}

	stopSpinnerFn := pleaseWait(ctx, "Logging in to Slack, it will take 25-40 seconds")
	defer stopSpinnerFn()
	var loginErr error
	sp.Token, sp.Cookie, loginErr = cl.Headless(ctx, username, password, stopSpinnerFn)
	if loginErr != nil {
		return sp, loginErr
	}
	return
}

func qrFlow(ctx context.Context, cl *slackauth.Client, ui browserAuthUIExt) (sp simpleProvider, err error) {
	imageData, err := ui.RequestQR(ctx, os.Stdout)
	if err != nil {
		return sp, err
	}
	tok, cook, err := cl.QRAuth(ctx, imageData)
	if err != nil {
		return sp, err
	}
	return simpleProvider{
		Token:  tok,
		Cookie: cook,
	}, nil
}
