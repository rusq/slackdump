package auth

import (
	"net/http"

	cookiemonster "github.com/MercuryEngineering/CookieMonster"
)

var _ Provider = FileCreds{}

type FileCreds struct {
	simpleProvider
}

// NewFileCreds creates new auth provider from token and Mozilla cookie file.
func NewFileCreds(token string, cookieFile string) (FileCreds, error) {
	if token == "" {
		return FileCreds{}, ErrNoToken
	}
	ptrCookies, err := cookiemonster.ParseFile(cookieFile)
	if err != nil {
		return FileCreds{}, err
	}
	fc := FileCreds{
		simpleProvider: simpleProvider{
			token:   token,
			cookies: deRefCookies(ptrCookies),
		},
	}
	return fc, nil
}

func deRefCookies(cc []*http.Cookie) []http.Cookie {
	var ret = make([]http.Cookie, len(cc))
	for i := range cc {
		ret[i] = *cc[i]
	}
	return ret
}
