package auth

import (
	"strings"
	"time"

	"github.com/rusq/slackdump/v3/auth/browser"
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

// RODWithRODHeadlessTimeout sets the timeout for the headless browser
// interaction.  It is a net time of headless browser interaction, without the
// browser starting time.
func RODWithRODHeadlessTimeout(d time.Duration) Option {
	return func(o *options) {
		if d <= 0 {
			return
		}
		o.rodOpts.autoTimeout = d
	}
}

// RODWithUserAgent sets the user agent string for the headless browser.
func RODWithUserAgent(ua string) Option {
	return func(o *options) {
		if ua != "" {
			o.rodOpts.userAgent = ua
		}
	}
}
