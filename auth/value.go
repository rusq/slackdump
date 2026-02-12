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
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

const (
	defaultPath   = "/"
	defaultDomain = ".slack.com"
)

var _ Provider = &ValueAuth{}

// ValueAuth stores Slack credentials.
type ValueAuth struct {
	simpleProvider
}

func NewValueAuth(token string, cookie string) (ValueAuth, error) {
	if token == "" {
		return ValueAuth{}, ErrNoToken
	}
	c := ValueAuth{simpleProvider{
		Token: token,
	}}
	if IsClientToken(token) {
		if len(cookie) == 0 {
			return ValueAuth{}, ErrNoCookies
		}
		c.Cookie = []*http.Cookie{
			makeCookie("d", cookie),
			makeCookie("d-s", fmt.Sprintf("%d", time.Now().Unix()-10)),
		}
	}
	return c, nil
}

func NewValueCookiesAuth(token string, cookies []*http.Cookie) (ValueAuth, error) {
	if token == "" {
		return ValueAuth{}, ErrNoToken
	}
	var found bool
	for _, c := range cookies {
		if c.Name == "d" {
			found = true
			break
		}
	}
	if !found {
		return ValueAuth{}, ErrNoCookies
	}
	return ValueAuth{simpleProvider{
		Token:  token,
		Cookie: cookies,
	}}, nil
}

// NewCookieOnlyAuth uses workspace name and dCookie to get the token value and returns
// a ValueAuth.
func NewCookieOnlyAuth(ctx context.Context, workspace, dCookie string) (ValueAuth, error) {
	if dCookie == "" {
		return ValueAuth{}, ErrNoCookies
	}
	token, cookies, err := getTokenByCookie(ctx, workspace, dCookie)
	if err != nil {
		return ValueAuth{}, err
	}
	return NewValueCookiesAuth(token, cookies)
}

var timeFunc = time.Now

func makeCookie(key, val string) *http.Cookie {
	if !urlsafe(val) {
		val = url.QueryEscape(val)
	}
	return &http.Cookie{
		Name:    key,
		Value:   val,
		Path:    defaultPath,
		Domain:  defaultDomain,
		Expires: timeFunc().AddDate(10, 0, 0).UTC(),
		Secure:  true,
	}
}

var reURLsafe = regexp.MustCompile(`[-._~%a-zA-Z0-9]+`)

func urlsafe(s string) bool {
	// https://www.ietf.org/rfc/rfc3986.txt
	st := reURLsafe.ReplaceAllString(s, "") // workaround for inability to use `(?!...)`
	return len(st) == 0
}
