package auth

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/rusq/slackdump/v3/auth/auth_ui"
	"github.com/rusq/slackdump/v3/auth/browser"
	"github.com/rusq/slackdump/v3/internal/structures"
)

var _ Provider = PlaywrightAuth{}
var defaultFlow = &auth_ui.Huh{}

// PlaywrightAuth is the playwright browser authentication provider.
//
// Deprecated: Use the [RodAuth] provider instead.
type PlaywrightAuth struct {
	simpleProvider
	opts options
}

type playwrightOptions struct {
	browser      browser.Browser
	flow         BrowserAuthUI
	loginTimeout time.Duration
	verbose      bool
}

type BrowserAuthUI interface {
	// RequestWorkspace should request the workspace name from the user.
	RequestWorkspace(w io.Writer) (string, error)
	// Stop indicates that the auth flow should cleanup and exit, if it is
	// keeping the state.
	Stop()
}

func NewPlaywrightAuth(ctx context.Context, opts ...Option) (PlaywrightAuth, error) {
	var br = PlaywrightAuth{
		opts: options{
			playwrightOptions: playwrightOptions{
				flow:         defaultFlow,
				browser:      browser.Bfirefox,
				loginTimeout: browser.DefLoginTimeout,
			},
		},
	}
	for _, opt := range opts {
		opt(&br.opts)
	}
	if IsDocker() {
		return PlaywrightAuth{}, &Error{Err: ErrNotSupported, Msg: "browser auth is not supported in docker, use token/cookie auth instead"}
	}

	if br.opts.workspace == "" {
		var err error
		br.opts.workspace, err = br.opts.flow.RequestWorkspace(os.Stdout)
		if err != nil {
			return br, err
		}
		defer br.opts.flow.Stop()
	}
	if wsp, err := structures.ExtractWorkspace(br.opts.workspace); err != nil {
		return br, err
	} else {
		br.opts.workspace = wsp
	}
	slog.Info("Please wait while Playwright is initialising.")
	slog.Info("If you're running it for the first time, it will take a couple of minutes...")
	stopSpinner := pleaseWait(ctx, "Initialising Playwright...")
	defer stopSpinner()
	auther, err := browser.New(br.opts.workspace, browser.OptBrowser(br.opts.browser), browser.OptTimeout(br.opts.loginTimeout), browser.OptVerbose(br.opts.verbose))
	if err != nil {
		return br, err
	}
	stopSpinner()

	token, cookies, err := auther.Authenticate(ctx)
	if err != nil {
		return br, err
	}
	br.simpleProvider = simpleProvider{
		Token:  token,
		Cookie: cookies,
	}
	return br, nil
}
