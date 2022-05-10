package browser

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/rusq/dlog"
)

// Client is the client for Browser Auth Provider.
type Client struct {
	workspace string
}

// New create new browser based client
func New(workspace string) (*Client, error) {
	if workspace == "" {
		return nil, errors.New("workspace can't be empty")
	}
	return &Client{workspace: workspace}, nil
}

func (cl *Client) Authenticate() (string, []http.Cookie, error) {
	pw, err := playwright.Run()
	if err != nil {
		return "", nil, err
	}
	defer pw.Stop()

	opts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	}
	browser, err := pw.Chromium.Launch(opts)
	if err != nil {
		return "", nil, err
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		return "", nil, err
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return "", nil, err
	}

	uri := fmt.Sprintf("https://%s.slack.com", cl.workspace)
	dlog.Printf("opening browser URL=%s", uri)

	if _, err := page.Goto(uri); err != nil {
		return "", nil, err
	}

	r := page.WaitForRequest(uri + "/api/api.features*")
	token, err := extractToken(r.URL())
	if err != nil {
		return "", nil, err
	}

	state, err := context.StorageState()
	if err != nil {
		return "", nil, err
	}
	if len(state.Cookies) == 0 {
		return "", nil, errors.New("empty cookies")
	}

	return token, convertCookies(state.Cookies), nil
}

// tokenRE is the regexp that matches a valid Slack Client token.
var tokenRE = regexp.MustCompile(`xoxc-[0-9]+-[0-9]+-[0-9]+-[0-9a-z]{64}`)

func extractToken(uri string) (string, error) {
	p, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return "", err
	}
	q := p.Query()
	token := q.Get("token")
	if token == "" {
		return "", errors.New("token not found")
	}
	if !tokenRE.MatchString(token) {
		return "", errors.New("invalid token value")
	}
	return token, nil
}

func convertCookies(pwc []playwright.Cookie) []http.Cookie {
	var ret = make([]http.Cookie, len(pwc))
	for i, p := range pwc {
		ret[i] = http.Cookie{
			Name:     p.Name,
			Value:    p.Value,
			Path:     p.Path,
			Domain:   p.Domain,
			Expires:  float2time(p.Expires),
			MaxAge:   0,
			Secure:   p.Secure,
			HttpOnly: p.HttpOnly,
			SameSite: sameSite(p.SameSite),
		}
	}
	return ret
}

var str2samesite = map[string]http.SameSite{
	"":       http.SameSiteDefaultMode,
	"Lax":    http.SameSiteLaxMode,
	"None":   http.SameSiteNoneMode,
	"Strict": http.SameSiteStrictMode,
}

// sameSite returns the constant value that maps to the string value of SameSite.
func sameSite(val string) http.SameSite {
	return str2samesite[val]
}

// float2time converts a float value of Unix time to time, nanoseconds value
// is discarded.  If v == -1, it returns the date approximately 5 years from
// Now().
func float2time(v float64) time.Time {
	if v == -1.0 {
		return time.Now().Add(5 * 365 * 24 * time.Hour)
	}
	return time.Unix(int64(v), 0)
}
