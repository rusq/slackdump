package auth

import (
	"encoding/json"
	"errors"
	"io"
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
	Token  string
	Cookie []http.Cookie
}

func (c simpleProvider) Validate() error {
	if c.Token == "" {
		return ErrNoToken
	}
	if len(c.Cookie) == 0 {
		return ErrNoCookies
	}
	return nil
}

func (c simpleProvider) SlackToken() string {
	return c.Token
}

func (c simpleProvider) Cookies() []http.Cookie {
	return c.Cookie
}

// deref dereferences []*T to []T.
func deref[T any](cc []*T) []T {
	var ret = make([]T, len(cc))
	for i := range cc {
		ret[i] = *cc[i]
	}
	return ret
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

	var s = simpleProvider{
		Token:  p.SlackToken(),
		Cookie: p.Cookies(),
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(s); err != nil {
		return err
	}

	return nil
}
