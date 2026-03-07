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
