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
// Package browser provides the playwright browser authentication provider.
//
// Deprecated: Use the [auth.RodAuth] provider instead.
package browser

import (
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// DefLoginTimeout is the default Slack login timeout
const DefLoginTimeout = 5 * time.Minute

//go:generate stringer -type Browser -trimprefix=B browser.go
type Browser int

const (
	Bfirefox Browser = iota
	Bchromium
)

type Option func(*Client)

func OptBrowser(b Browser) Option {
	return func(c *Client) {
		if b < Bfirefox || Bchromium < b {
			b = Bfirefox
		}
		c.br = b
	}
}

func OptTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d < 0 {
			return
		}
		c.loginTimeout = float64(d.Milliseconds())
	}
}

func OptVerbose(b bool) Option {
	return func(c *Client) {
		c.verbose = b
	}
}

func (e *Browser) Set(v string) error {
	v = strings.ToLower(v)
	for i := 0; i < len(_Browser_index)-1; i++ {
		if strings.ToLower(_Browser_name[_Browser_index[i]:_Browser_index[i+1]]) == v {
			*e = Browser(i)
			return nil
		}
	}
	var allowed []string
	for i := 0; i < len(_Browser_index)-1; i++ {
		allowed = append(allowed, _Browser_name[_Browser_index[i]:_Browser_index[i+1]])
	}
	return fmt.Errorf("unknown browser: %s, allowed: %v", v, allowed)
}

// client returns the appropriate client from playwright.Playwright.
func (br Browser) client(pw *playwright.Playwright) playwright.BrowserType {
	switch br {
	default:
		fallthrough
	case Bfirefox:
		return pw.Firefox
	case Bchromium:
		return pw.Chromium
	}
	// unreachable
}
