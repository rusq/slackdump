package auth

import (
	"context"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rusq/slackdump/v2/auth/auth_ui"
	"github.com/rusq/slackdump/v2/auth/browser"
)

var _ Provider = BrowserAuth{}
var defaultFlow = &auth_ui.Survey{}

type BrowserAuth struct {
	simpleProvider
	opts browserOpts
}

type browserOpts struct {
	workspace    string
	browser      browser.Browser
	flow         BrowserAuthUI
	loginTimeout time.Duration
	verbose      bool
}

type BrowserAuthUI interface {
	RequestWorkspace(w io.Writer) (string, error)
	Stop()
}

func NewBrowserAuth(ctx context.Context, opts ...Option) (BrowserAuth, error) {
	var br = BrowserAuth{
		opts: browserOpts{
			flow:         defaultFlow,
			browser:      browser.Bfirefox,
			loginTimeout: browser.DefLoginTimeout,
		},
	}
	for _, opt := range opts {
		opt(&options{browserOpts: &br.opts})
	}
	if isDocker() {
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
	if wsp, err := sanitize(br.opts.workspace); err != nil {
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

func sanitize(workspace string) (string, error) {
	if !strings.Contains(workspace, ".slack.com") {
		return workspace, nil
	}
	if strings.HasPrefix(workspace, "https://") {
		uri, err := url.Parse(workspace)
		if err != nil {
			return "", err
		}
		workspace = uri.Host
	}
	// parse
	parts := strings.Split(workspace, ".")
	return parts[0], nil
}

func isDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}
