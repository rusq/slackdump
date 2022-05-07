package auth

import (
	"fmt"
	"net/http"
	"time"
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

func makeCookie(key, val string) http.Cookie {
	return http.Cookie{
		Name:    key,
		Value:   val,
		Path:    "/",
		Domain:  ".slack.com",
		Expires: time.Now().AddDate(10, 0, 0),
		Secure:  true,
	}
}
