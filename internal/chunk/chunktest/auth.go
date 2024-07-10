package chunktest

import (
	"context"
	"net/http"

	"github.com/rusq/slack"
)

// TestAuth to use with the chunktest server.
type TestAuth struct {
	FakeToken            string
	FakeCookies          []*http.Cookie
	WantValidateError    error
	WantTestError        error
	WantHTTPClient       *http.Client
	WantHTTPClientErr    error
	WantAuthTestResponse *slack.AuthTestResponse
}

// SlackToken should return the Slack Token value.
func (a *TestAuth) SlackToken() string {
	return a.FakeToken
}

// Cookies should return a set of Slack Session cookies.
func (a *TestAuth) Cookies() []*http.Cookie {
	return nil
}

// Validate should return error, in case the token or cookies cannot be
// retrieved.
func (a *TestAuth) Validate() error {
	// chur
	return a.WantValidateError
}

// Test tests if credentials are valid.
func (a *TestAuth) Test(ctx context.Context) (*slack.AuthTestResponse, error) {
	return a.WantAuthTestResponse, a.WantTestError
}

// HTTPClient returns an authenticated HTTP client
func (a *TestAuth) HTTPClient() (*http.Client, error) {
	return a.WantHTTPClient, a.WantHTTPClientErr
}
