package browser

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
)

//go:generate mockgen -package browser -destination playwright_test.go github.com/playwright-community/playwright-go Request

// tokenRE is the regexp that matches a valid Slack Client token.
var tokenRE = regexp.MustCompile(`xoxc-[0-9]+-[0-9]+-[0-9]+-[0-9a-z]{64}`)

const maxMultipartMem = 65536

var (
	ErrNoToken            = errors.New("no token found")
	ErrInvalidTokenValue  = errors.New("invalid token value")
	ErrInvalidContentType = errors.New("invalid content-type header")
)

// extractToken extracts token from the request.
func extractToken(r playwright.Request) (string, error) {
	if r == nil {
		return "", errors.New("no request")
	}
	if r.Method() == http.MethodGet {
		return extractTokenGet(r.URL())
	} else if r.Method() == http.MethodPost {
		return extractTokenPost(r)
	}
	return "", errors.New("invalid request method")
}

// extractTokenGet extracts token from the query string.
func extractTokenGet(uri string) (string, error) {
	p, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return "", err
	}
	q := p.Query()
	token := q.Get("token")
	if token == "" {
		return "", ErrNoToken
	}
	if !tokenRE.MatchString(token) {
		return "", ErrInvalidTokenValue
	}
	return token, nil
}

// extractTokenPost extracts token from the request body.
func extractTokenPost(r playwright.Request) (string, error) {
	boundary, err := boundary(r)
	if err != nil {
		return "", err
	}
	data, err := r.PostData()
	if err != nil {
		return "", err
	}
	return tokenFromMultipart(data, boundary)
}

// tokenFromMultipart extracts token from the multipart form.
func tokenFromMultipart(s string, boundary string) (string, error) {
	mp := multipart.NewReader(strings.NewReader(s), boundary)
	form, err := mp.ReadForm(maxMultipartMem)
	if err != nil {
		return "", err
	}
	tok, ok := form.Value["token"]
	if !ok {
		return "", errors.New("token not found")
	}
	if len(tok) != 1 {
		return "", errors.New("invalid token value")
	}
	if !tokenRE.MatchString(tok[0]) {
		return "", errors.New("invalid token value")
	}
	return tok[0], nil
}

// boundary extracts boundary from the request.
func boundary(r playwright.Request) (string, error) {
	v, err := r.HeaderValue("Content-Type")
	if err != nil {
		return "", err
	}
	values := strings.Split(v, ",")
	if len(values) != 1 {
		return "", ErrInvalidContentType
	}
	contentType, boundary, found := strings.Cut(values[0], "; boundary=")
	if !found || contentType != "multipart/form-data" {
		return "", ErrInvalidContentType
	}
	return boundary, nil
}
