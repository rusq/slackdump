package auth

import (
	"fmt"
	"net/http"
	"time"
)

var _ Provider = &ValueCreds{}

// ValueCreds stores Slack credentials.
type ValueCreds struct {
	simpleProvider
}

func NewValueCreds(token string, cookie string) (ValueCreds, error) {
	if token == "" {
		return ValueCreds{}, ErrNoToken
	}
	if cookie == "" {
		return ValueCreds{}, ErrNoCookies
	}
	return ValueCreds{simpleProvider{
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
