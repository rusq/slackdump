// Package chttp (Cooked HTTP) provides a wrapper around http.Client with
// cookies.
package chttp

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"golang.org/x/net/publicsuffix"
)

// NewWithTransport inits the HTTP client with cookies.  It allows to use
// the custom Transport.
func NewWithTransport(cookieDomain string, cookies []*http.Cookie, rt http.RoundTripper) *http.Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	url, err := url.Parse(cookieDomain)
	if err != nil {
		panic(err) //shouldn't happen
	}
	jar.SetCookies(url, cookies)
	cl := http.Client{
		Jar:       jar,
		Transport: rt,
	}
	return &cl
}

// New returns the HTTP client with cookies and default transport.
func New(cookieDomain string, cookies []*http.Cookie) *http.Client {
	return NewWithTransport(cookieDomain, cookies, NewTransport(nil))
}

func sliceOfPtr[T any](cc []T) []*T {
	var ret = make([]*T, len(cc))
	for i := range cc {
		ret[i] = &cc[i]
	}
	return ret
}

func ConvertCookies(cc []http.Cookie) []*http.Cookie {
	return sliceOfPtr(cc)
}
