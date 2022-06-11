package auth

import (
	"errors"
	"net/http"
)

// Provider is the Slack Authentication provider.
type Provider interface {
	// SlackToken should return the Slack Token value.
	SlackToken() string
	// Cookies should returns a set of Slack Session cookies.
	Cookies() []http.Cookie
	// Validate should return error, in case the token or cookies cannot be
	// retrieved.
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
