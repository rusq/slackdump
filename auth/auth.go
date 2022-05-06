package auth

import (
	"errors"
	"net/http"
)

type Provider interface {
	SlackToken() string
	Cookies() []http.Cookie
	Validate() error
}

var (
	ErrNoToken   = errors.New("no token")
	ErrNoCookies = errors.New("no cookies")
)

type simpleProvider struct {
	token   string
	cookies []http.Cookie
}

func (c simpleProvider) Validate() error {
	if c.token == "" {
		return ErrNoToken
	}
	if len(c.cookies) == 0 {
		return ErrNoCookies
	}
	return nil
}

func (c simpleProvider) SlackToken() string {
	return c.token
}

func (c simpleProvider) Cookies() []http.Cookie {
	return c.cookies
}
