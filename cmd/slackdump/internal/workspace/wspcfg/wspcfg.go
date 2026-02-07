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
// Package wspcfg contains workspace configuration variables.
package wspcfg

import (
	"flag"
	"time"

	"github.com/rusq/osenv/v2"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/auth/browser"
)

var (
	SlackToken      string
	SlackCookie     string
	LoginTimeout    time.Duration = browser.DefLoginTimeout // overall login time.
	HeadlessTimeout time.Duration = auth.RODHeadlessTimeout // net interaction time.
	RODUserAgent    string                                  // when empty, slackauth uses the default user agent.
	// playwright stuff
	Browser       browser.Browser
	LegacyBrowser bool
)

func SetWspFlags(fs *flag.FlagSet) {
	fs.StringVar(&SlackToken, "token", osenv.Secret("SLACK_TOKEN", ""), "Slack `token`")
	fs.StringVar(&SlackCookie, "cookie", osenv.Secret("SLACK_COOKIE", ""), "d= cookie `value` or a path to a cookie.txt file\n(environment: SLACK_COOKIE)")
	fs.Var(&Browser, "browser", "browser to use for legacy EZ-Login 3000 (default: firefox)")
	fs.DurationVar(&LoginTimeout, "browser-timeout", LoginTimeout, "Browser login `timeout`")
	fs.DurationVar(&HeadlessTimeout, "autologin-timeout", HeadlessTimeout, "headless autologin `timeout`, without the browser starting time, just the interaction time")
	fs.BoolVar(&LegacyBrowser, "legacy-browser", false, "use legacy browser automation (playwright) for EZ-Login 3000")
	fs.StringVar(&RODUserAgent, "user-agent", "", "override the user agent string for EZ-Login 3000")
}
