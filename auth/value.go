package auth

import (
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
	if cookie == "" {
		return ValueAuth{}, ErrNoCookies
	}
	return ValueAuth{simpleProvider{
		token: token,
		cookies: []http.Cookie{
			makeCookie("d", cookie),
			makeCookie("d-s", fmt.Sprintf("%d", time.Now().Unix()-10)),
		},
	}}, nil
}

func (ValueAuth) Type() Type {
	return TypeValue
}

var timeFunc = time.Now

func makeCookie(key, val string) http.Cookie {
	if !urlsafe(val) {
		val = url.QueryEscape(val)
	}
	return http.Cookie{
		Name:    key,
		Value:   val,
		Path:    defaultPath,
		Domain:  defaultDomain,
		Expires: timeFunc().AddDate(10, 0, 0),
		Secure:  true,
	}
}

var reURLsafe = regexp.MustCompile(`[-._~%a-zA-Z0-9]+`)

func urlsafe(s string) bool {
	// https://www.ietf.org/rfc/rfc3986.txt
	st := reURLsafe.ReplaceAllString(s, "") // workaround for inability to use `(?!...)`
	return len(st) == 0
}
