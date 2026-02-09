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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"runtime/trace"

	"github.com/charmbracelet/huh/spinner"
	"github.com/rusq/chttp"
	"github.com/rusq/slack"
	"github.com/rusq/slackauth"
)

const SlackURL = "https://slack.com"

// tokenRE is the regexp that matches a valid Slack Client token.
var tokenRE = regexp.MustCompile(`xoxc-[0-9]+-[0-9]+-[0-9]+-[0-9a-z]{64}`)

// Provider is the Slack Authentication provider.
//
//go:generate mockgen -destination ../internal/mocks/mock_auth/mock_auth.go github.com/rusq/slackdump/v4/auth Provider
type Provider interface {
	// SlackToken should return the Slack Token value.
	SlackToken() string
	// Cookies should return a set of Slack Session cookies.
	Cookies() []*http.Cookie
	// Validate should return error, in case the token or cookies cannot be
	// retrieved.
	Validate() error
	// Test tests if credentials are valid.
	Test(ctx context.Context) (*slack.AuthTestResponse, error)
	// Client returns an authenticated HTTP client
	HTTPClient() (*http.Client, error)
}

var (
	ErrNoToken      = errors.New("no token")
	ErrNoCookies    = errors.New("no cookies")
	ErrNotSupported = errors.New("not supported")
	// ErrCancelled may be returned by auth providers, if the authentication
	// process was cancelled.
	ErrCancelled = errors.New("authentication cancelled")
)

type simpleProvider struct {
	Token  string
	Cookie []*http.Cookie
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

func (c simpleProvider) Cookies() []*http.Cookie {
	return c.Cookie
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

	s := simpleProvider{
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
	return tokenRE.MatchString(tok)
}

// TestAuth attempts to authenticate with the given provider.  It will return
// AuthError if failed.
func (s simpleProvider) Test(ctx context.Context) (*slack.AuthTestResponse, error) {
	ctx, task := trace.NewTask(ctx, "TestAuth")
	defer task.End()

	httpCl, err := s.HTTPClient()
	if err != nil {
		return nil, &Error{Err: err}
	}
	cl := slack.New(s.Token, slack.OptionHTTPClient(httpCl))

	region := trace.StartRegion(ctx, "simpleProvider.Test")
	defer region.End()
	ai, err := cl.AuthTestContext(ctx)
	if err != nil {
		return ai, &Error{Err: err}
	}
	return ai, nil
}

func (s simpleProvider) HTTPClient() (*http.Client, error) {
	return chttp.New(SlackURL, s.Cookies(), chttp.WithUserAgent(slackauth.DefaultUserAgent))
}

func pleaseWait(ctx context.Context, msg string) func() {
	sctx, stopSpinner := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = spinner.New().
			Type(spinner.Dots).
			Title(msg).
			Context(sctx).
			Run()
	}()

	return func() {
		stopSpinner()
		<-done
	}
}
