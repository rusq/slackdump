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
