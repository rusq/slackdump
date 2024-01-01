package auth

import (
	"strings"
	"time"

	"github.com/rusq/slackdump/v2/auth/browser"
)

type options struct {
	browserOpts
	rodOpts
	workspace string
}

type Option func(*options)

func BrowserWithAuthFlow(flow BrowserAuthUI) Option {
	return func(o *options) {
		if flow == nil {
			return
		}
		o.browserOpts.flow = flow
	}
}

func BrowserWithWorkspace(name string) Option {
	return func(o *options) {
		o.workspace = strings.ToLower(name)
	}
}

func BrowserWithBrowser(b browser.Browser) Option {
	return func(o *options) {
		o.browserOpts.browser = b
	}
}

func BrowserWithTimeout(d time.Duration) Option {
	return func(o *options) {
		if d < 0 {
			return
		}
		o.browserOpts.loginTimeout = d
	}
}

func BrowserWithVerbose(b bool) Option {
	return func(o *options) {
		o.browserOpts.verbose = b
	}
}
