package auth

import (
	cookiemonster "github.com/MercuryEngineering/CookieMonster"
)

var _ Provider = CookieFileAuth{}

type CookieFileAuth struct {
	simpleProvider
}

// NewCookieFileAuth creates new auth provider from token and Mozilla cookie file.
func NewCookieFileAuth(token string, cookieFile string) (CookieFileAuth, error) {
	if token == "" {
		return CookieFileAuth{}, ErrNoToken
	}
	ptrCookies, err := cookiemonster.ParseFile(cookieFile)
	if err != nil {
		return CookieFileAuth{}, err
	}
	fc := CookieFileAuth{
		simpleProvider: simpleProvider{
			Token:  token,
			Cookie: ptrCookies,
		},
	}
	return fc, nil
}

func (CookieFileAuth) Type() Type {
	return TypeCookieFile
}
