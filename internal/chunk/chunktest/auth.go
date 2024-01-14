package chunktest

import (
	"context"
	"net/http"

	"github.com/rusq/slackdump/v3/auth"
)

// TestAuth to use with the chunktest server.
type TestAuth struct {
	FakeToken         string
	FakeCookies       []*http.Cookie
	WantValidateError error
	WantTestError     error
	WantHTTPClient    *http.Client
	WantHTTPClientErr error
}

// SlackToken should return the Slack Token value.
func (a *TestAuth) SlackToken() string {
	return a.FakeToken
}

// Cookies should return a set of Slack Session cookies.
func (a *TestAuth) Cookies() []*http.Cookie {
	return nil
}

// Type returns the auth type.
func (a *TestAuth) Type() auth.Type {
	return auth.Type(255)
}

// Validate should return error, in case the token or cookies cannot be
// retrieved.
func (a *TestAuth) Validate() error {
	// chur
	return a.WantValidateError
}

// Test tests if credentials are valid.
func (a *TestAuth) Test(ctx context.Context) error {
	return a.WantTestError
}

// Client returns an authenticated HTTP client
func (a *TestAuth) HTTPClient() (*http.Client, error) {
	return a.WantHTTPClient, a.WantHTTPClientErr
}
