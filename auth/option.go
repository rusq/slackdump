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
	"strings"
	"time"

	"github.com/rusq/slackdump/v4/auth/browser"
)

type options struct {
	playwrightOptions
	rodOpts
	workspace string
}

type Option func(*options)

func BrowserWithAuthFlow(flow BrowserAuthUI) Option {
	return func(o *options) {
		if flow == nil {
			return
		}
		o.playwrightOptions.flow = flow
	}
}

func BrowserWithWorkspace(name string) Option {
	return func(o *options) {
		o.workspace = strings.ToLower(name)
	}
}

func BrowserWithBrowser(b browser.Browser) Option {
	return func(o *options) {
		o.playwrightOptions.browser = b
	}
}

func BrowserWithTimeout(d time.Duration) Option {
	return func(o *options) {
		if d < 0 {
			return
		}
		o.playwrightOptions.loginTimeout = d
	}
}

func BrowserWithVerbose(b bool) Option {
	return func(o *options) {
		o.playwrightOptions.verbose = b
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
