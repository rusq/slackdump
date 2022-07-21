package auth

import (
	"errors"
	"net/http"
)

// Type is the auth type.
type Type uint8

// All supported auth types.
const (
	TypeInvalid Type = iota
	TypeValue
	TypeCookieFile
	TypeBrowser
)

// Provider is the Slack Authentication provider.
type Provider interface {
	// SlackToken should return the Slack Token value.
	SlackToken() string
	// Cookies should returns a set of Slack Session cookies.
	Cookies() []http.Cookie
	// Type returns the auth type.
	Type() Type
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

// deref dereferences []*T to []T.
func deref[T any](cc []*T) []T {
	var ret = make([]T, len(cc))
	for i := range cc {
		ret[i] = *cc[i]
	}
	return ret
}
