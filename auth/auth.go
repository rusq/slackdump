package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime/trace"
	"strings"

	"github.com/rusq/slackdump/v2/internal/chttp"
	"github.com/slack-go/slack"
)

// Type is the auth type.
//
//go:generate stringer -type Type -linecomment
type Type uint8

// All supported auth types.
const (
	TypeInvalid    Type = iota // Invalid
	TypeValue                  // Value
	TypeCookieFile             // Cookie File
	TypeBrowser                // EZ-Login 3000
)

// Provider is the Slack Authentication provider.
//
//go:generate mockgen -destination ../internal/mocks/mock_auth/mock_auth.go github.com/rusq/slackdump/v2/auth Provider
type Provider interface {
	// SlackToken should return the Slack Token value.
	SlackToken() string
	// Cookies should return a set of Slack Session cookies.
	Cookies() []http.Cookie
	// Type returns the auth type.
	Type() Type
	// Validate should return error, in case the token or cookies cannot be
	// retrieved.
	Validate() error
	// Test tests if credentials are valid.
	Test(ctx context.Context) error
}

var (
	ErrNoToken   = errors.New("no token")
	ErrNoCookies = errors.New("no cookies")
)

type simpleProvider struct {
	Token  string
	Cookie []http.Cookie
}

func (c simpleProvider) Validate() error {
	if c.Token == "" {
		return ErrNoToken
	}
	if IsClientToken(c.Token) && len(c.Cookie) == 0 {
		return ErrNoCookies
	}
	return nil
}

func (c simpleProvider) SlackToken() string {
	return c.Token
}

func (c simpleProvider) Cookies() []http.Cookie {
	return c.Cookie
}

// deref dereferences []*T to []T.
func deref[T any](cc []*T) []T {
	var ret = make([]T, len(cc))
	for i := range cc {
		ret[i] = *cc[i]
	}
	return ret
}

func ref[T any](cc []T) []*T {
	var ret = make([]*T, len(cc))
	for i := range cc {
		ret[i] = &cc[i]
	}
	return ret
}

// Load deserialises JSON data from reader and returns a ValueAuth, that can
// be used to authenticate Slackdump.  It will return ErrNoToken or
// ErrNoCookie if the authentication information is missing.
func Load(r io.Reader) (ValueAuth, error) {
	dec := json.NewDecoder(r)
	var s simpleProvider
	if err := dec.Decode(&s); err != nil {
		return ValueAuth{}, err
	}
	return ValueAuth{s}, s.Validate()
}

// Save serialises authentication information to writer.  It will return
// ErrNoToken or ErrNoCookie if provider fails validation.
func Save(w io.Writer, p Provider) error {
	if err := p.Validate(); err != nil {
		return err
	}

	var s = simpleProvider{
		Token:  p.SlackToken(),
		Cookie: p.Cookies(),
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(s); err != nil {
		return err
	}

	return nil
}

// IsClientToken returns true if the tok is a web-client token.
func IsClientToken(tok string) bool {
	return strings.HasPrefix(tok, "xoxc-")
}

// TestAuth attempts to authenticate with the given provider.  It will return
// AuthError if failed.
func (s simpleProvider) Test(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "TestAuth")
	defer task.End()

	httpCl := chttp.New("https://slack.com", chttp.ConvertCookies(s.Cookie))

	cl := slack.New(s.Token, slack.OptionHTTPClient(httpCl))

	region := trace.StartRegion(ctx, "AuthTestContext")
	defer region.End()
	_, err := cl.AuthTestContext(ctx)
	if err != nil {
		return &Error{Err: err}
	}
	return nil
}
