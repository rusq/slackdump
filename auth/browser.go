package auth

import (
	"context"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"

	"github.com/rusq/slackdump/v2/auth/auth_ui"
	"github.com/rusq/slackdump/v2/auth/browser"
)

var _ Provider = BrowserAuth{}
var defaultFlow = &auth_ui.CLI{}

type BrowserAuth struct {
	simpleProvider
	flow      BrowserAuthUI
	workspace string
}

type BrowserAuthUI interface {
	RequestWorkspace(w io.Writer) (string, error)
	Stop()
}

type BrowserOption func(*BrowserAuth)

func BrowserWithAuthFlow(flow BrowserAuthUI) BrowserOption {
	return func(ba *BrowserAuth) {
		if flow == nil {
			return
		}
		ba.flow = flow
	}
}

func BrowserWithWorkspace(name string) BrowserOption {
	return func(ba *BrowserAuth) {
		ba.workspace = name
	}
}

func NewBrowserAuth(ctx context.Context, opts ...BrowserOption) (BrowserAuth, error) {
	var br = BrowserAuth{
		flow: defaultFlow,
	}
	for _, opt := range opts {
		opt(&br)
	}

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		return br, err
	}
	if br.workspace == "" {
		var err error
		br.workspace, err = br.flow.RequestWorkspace(os.Stdout)
		if err != nil {
			return br, err
		}
		defer br.flow.Stop()
	}
	if wsp, err := sanitize(br.workspace); err != nil {
		return br, err
	} else {
		br.workspace = wsp
	}

	auther, err := browser.New(br.workspace)
	if err != nil {
		return br, err
	}
	token, cookies, err := auther.Authenticate(ctx)
	if err != nil {
		return br, err
	}
	br.simpleProvider = simpleProvider{
		token:   token,
		cookies: cookies,
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
