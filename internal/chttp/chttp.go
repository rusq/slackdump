// Package chttp provides some convenience function to wrap the standard http
// Client.
package chttp

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"golang.org/x/net/publicsuffix"
)

// New inits the HTTP client with cookies.
func New(cookieDomain string, cookies []*http.Cookie, rt http.RoundTripper) *http.Client {
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

// NewWithToken returns the HTTP client with cookies, that augments requests
// with slack token.
func NewWithToken(token string, cookieDomain string, cookies []*http.Cookie) *http.Client {
	tr := NewTransport(nil)
	tr.BeforeReq = func(req *http.Request) {
		// req.V
		// if req.Method == http.MethodGet {
		// 	req.Form.Add("token", token)
		// }
	}
	return New(cookieDomain, cookies, tr)
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
