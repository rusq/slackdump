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
