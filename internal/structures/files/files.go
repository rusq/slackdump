// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package files

import (
	"net/url"

	"github.com/rusq/slack"
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
		return UpdateAllLinks(f, func(s *string) error {
			var err error
			*s, err = addToken(*s, token)
			if err != nil {
				return err
			}
			return nil
		})
	}
}

// fileThumbLinks returns slice of pointers to all private URL links of the file.
func filePrivateLinks(f *slack.File) []*string {
	return []*string{
		&f.URLPrivate,
		&f.URLPrivateDownload,
	}
}

// fileThumbLinks returns slice of pointers to all thumbnail URLs of the file.
func fileThumbLinks(f *slack.File) []*string {
	return []*string{
		&f.Thumb64,
		&f.Thumb80,
		&f.Thumb160,
		&f.Thumb360,
		&f.Thumb360Gif,
		&f.Thumb480,
		&f.Thumb720,
		&f.Thumb960,
		&f.Thumb1024,
	}
}

// UpdateAllLinks calls fn with pointer to each file URL except permalinks.
// fn can modify the string pointed by ptrS.
func UpdateAllLinks(f *slack.File, fn func(ptrS *string) error) error {
	return apply(append(fileThumbLinks(f), filePrivateLinks(f)...), fn)
}

func UpdateDownloadLinks(f *slack.File, fn func(ptrS *string) error) error {
	return apply(filePrivateLinks(f), fn)
}

// UpdatePathFn sets the URLPrivate and URLPrivateDownload for the file at addr
// to the specified path.
func UpdatePathFn(path string) UpdateFunc {
	return func(f *slack.File) error {
		return UpdateDownloadLinks(f, func(ptrS *string) error {
			*ptrS = path
			return nil
		})
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

// apply calls fn for each element of slice elements.
func apply[T any](elements []*T, fn func(el *T) error) error {
	for _, ptr := range elements {
		if err := fn(ptr); err != nil {
			return err
		}
	}
	return nil
}
