package browser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/rusq/slackdump/v2/logger"
)

const slackDomain = ".slack.com"

// Client is the client for Browser Auth Provider.
type Client struct {
	workspace  string
	pageClosed chan bool // will receive a notification that the page is closed prematurely.
	br         Browser
}

var Logger logger.Interface = logger.Default

var installFn = playwright.Install

// New create new browser based client.
func New(workspace string, opts ...Option) (*Client, error) {
	if workspace == "" {
		return nil, errors.New("workspace can't be empty")
	}
	cl := &Client{workspace: workspace, pageClosed: make(chan bool, 1), br: Bfirefox}
	for _, opt := range opts {
		opt(cl)
	}
	if err := installFn(&playwright.RunOptions{
		Browsers: []string{cl.br.String()},
	}); err != nil {
		if !strings.Contains(err.Error(), "could not run driver") || runtime.GOOS == "windows" {
			return nil, err
		}
		if err := pwRepair(cl.br.String()); err != nil {
			return nil, err
		}
	}
	return cl, nil
}

func (cl *Client) Authenticate(ctx context.Context) (string, []*http.Cookie, error) {
	ctx, task := trace.NewTask(ctx, "Authenticate")
	defer task.End()

	var (
		_s = playwright.String
		_f = playwright.Float
		_b = playwright.Bool
	)

	pw, err := playwright.Run()
	if err != nil {
		return "", nil, err
	}
	defer pw.Stop()

	opts := playwright.BrowserTypeLaunchOptions{
		Headless: _b(false),
	}

	browser, err := cl.br.client(pw).Launch(opts)
	if err != nil {
		return "", nil, err
	}
	defer browser.Close()

	context, err := browser.NewContext()
	if err != nil {
		return "", nil, err
	}
	defer context.Close()

	// disable the "cookies" nag screen.
	if err := context.AddCookies(playwright.BrowserContextAddCookiesOptionsCookies{
		Domain:  _s(slackDomain),
		Path:    _s("/"),
		Name:    _s("OptanonAlertBoxClosed"),
		Value:   _s(time.Now().Add(-10 * time.Minute).Format(time.RFC3339)),
		Expires: _f(float64(time.Now().AddDate(0, 0, 30).Unix())),
	}); err != nil {
		return "", nil, err
	}

	page, err := context.NewPage()
	if err != nil {
		return "", nil, err
	}
	page.On("close", func() { trace.Log(ctx, "user", "page closed"); close(cl.pageClosed) })

	uri := fmt.Sprintf("https://%s"+slackDomain, cl.workspace)
	l().Debugf("opening browser URL=%s", uri)

	if _, err := page.Goto(uri); err != nil {
		return "", nil, err
	}

	var r playwright.Request
	if err := cl.withBrowserGuard(ctx, func() error {
		var err error
		r, err = page.WaitForRequest(uri + "/api/api.features*")
		return err
	}); err != nil {
		return "", nil, err
	}

	token, err := extractToken(r)
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

var ErrBrowserClosed = errors.New("browser closed or timed out")

func (cl *Client) withBrowserGuard(ctx context.Context, fn func() error) error {
	done := make(chan struct{})
	errC := make(chan error, 1)
	go func() {
		defer close(done)
		errC <- fn()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-cl.pageClosed:
		return ErrBrowserClosed
	case err := <-errC:
		return err
	case <-done:
	}
	return nil
}

func convertCookies(pwc []playwright.Cookie) []*http.Cookie {
	ret := make([]*http.Cookie, 0, len(pwc))
	for _, p := range pwc {
		ret = append(ret, &http.Cookie{
			Name:     p.Name,
			Value:    p.Value,
			Path:     p.Path,
			Domain:   p.Domain,
			Expires:  float2time(p.Expires),
			MaxAge:   0,
			Secure:   p.Secure,
			HttpOnly: p.HttpOnly,
			SameSite: sameSite(p.SameSite),
		})
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

func l() logger.Interface {
	if Logger == nil {
		return logger.Default
	}
	return Logger
}

// newDriverFn is the function that creates a new driver.  It is set to
// playwright.NewDriver by default, but can be overridden for testing.
var newDriverFn = playwright.NewDriver

// pwRepair attempts to repair the playwright installation.
func pwRepair(browser string) error {
	if browser == "" {
		return nil
	}
	drv, err := newDriverFn(&playwright.RunOptions{
		Browsers: []string{browser},
	})
	if err != nil {
		return err
	}

	// check node permissions
	if err := pwIsKnownProblem(drv.DriverDirectory); err != nil {
		return err
	}
	if err := os.RemoveAll(drv.DriverDirectory); err != nil {
		return err
	}

	// attempt to reinstall
	if err := installFn(&playwright.RunOptions{
		Browsers: []string{browser},
	}); err != nil {
		// we did everything we could, but it still failed.
		return err
	}
	return nil
}

var errUnknownProblem = errors.New("unknown problem")

// pwIsKnownProblem checks if the playwright installation is in a known
// problematic state, and if yes, return nil.  If the problem is unknown,
// returns an errUnknownProblem.
func pwIsKnownProblem(path string) error {
	if runtime.GOOS == "windows" {
		// this should not ever happen on windows, as this problem relates to
		// executable flag not being set, which is not a thing in a
		// DOS/Windows world.
		return errors.New("impossible has just happened, call the exorcist")
	}
	fi, err := os.Stat(filepath.Join(path, "node"))
	if err != nil {
		return err
	}
	// check if the file is executable, and if yes, return an error, because
	// we wouldn't know what to do.
	if fi.Mode()&0o111 != 0 {
		return errUnknownProblem
	}
	return nil
}
