package files

import (
	"net/url"

	"github.com/slack-go/slack"
)

// UpdateFunc is the signature of the function that modifies the file passed
// as an argument.
type UpdateFunc func(f *slack.File) error

// UpdateTokenFn returns a file update function that adds the t= query parameter
// with token value. If token value is empty, the function does nothing.
func UpdateTokenFn(token string) UpdateFunc {
	return func(f *slack.File) error {
		if token == "" {
			return nil
		}
		var err error
		update := func(s *string, t string) {
			if err != nil {
				return
			}
			*s, err = addToken(*s, t)
		}
		update(&f.URLPrivate, token)
		update(&f.URLPrivateDownload, token)
		update(&f.Thumb64, token)
		update(&f.Thumb80, token)
		update(&f.Thumb160, token)
		update(&f.Thumb360, token)
		update(&f.Thumb360Gif, token)
		update(&f.Thumb480, token)
		update(&f.Thumb720, token)
		update(&f.Thumb960, token)
		update(&f.Thumb1024, token)
		return nil
	}
}

// UpdatePathFn sets the URLPrivate and URLPrivateDownload for the file at addr
// to the specified path.
func UpdatePathFn(path string) UpdateFunc {
	return func(f *slack.File) error {
		f.URLPrivateDownload = path
		f.URLPrivate = path
		return nil
	}
}

// addToken updates the uri, adding the t= query parameter with token value.
// if token or url is empty, it does nothing.
func addToken(uri string, token string) (string, error) {
	if token == "" || uri == "" {
		return uri, nil
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	val := u.Query()
	val.Set("t", token)
	u.RawQuery = val.Encode()
	return u.String(), nil
}
