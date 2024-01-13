package auth

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rusq/slackdump/v2/auth/auth_ui"
	"github.com/rusq/slackdump/v2/auth/browser"
)

var _ Provider = BrowserAuth{}
var defaultFlow = &auth_ui.Huh{}

type BrowserAuth struct {
	simpleProvider
	opts options
}

type browserOpts struct {
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

func NewBrowserAuth(ctx context.Context, opts ...Option) (BrowserAuth, error) {
	var br = BrowserAuth{
		opts: options{
			browserOpts: browserOpts{
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
		return BrowserAuth{}, &Error{Err: ErrNotSupported, Msg: "browser auth is not supported in docker, use token/cookie auth instead"}
	}

	if br.opts.workspace == "" {
		var err error
		br.opts.workspace, err = br.opts.flow.RequestWorkspace(os.Stdout)
		if err != nil {
			return br, err
		}
		defer br.opts.flow.Stop()
	}
	if wsp, err := auth_ui.Sanitize(br.opts.workspace); err != nil {
		return br, err
	} else {
		br.opts.workspace = wsp
	}
	auther, err := browser.New(br.opts.workspace, browser.OptBrowser(br.opts.browser), browser.OptTimeout(br.opts.loginTimeout), browser.OptVerbose(br.opts.verbose))
	if err != nil {
		return br, err
	}
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

func (BrowserAuth) Type() Type {
	return TypeBrowser
}
