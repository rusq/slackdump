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
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_makeCookie(t *testing.T) {
	oldTimeFunc := timeFunc
	timeFunc = func() time.Time {
		return time.Date(2022, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	defer func() {
		timeFunc = oldTimeFunc
	}()

	type args struct {
		key string
		val string
	}
	tests := []struct {
		name string
		args args
		want *http.Cookie
	}{
		{
			"values are properly propagated",
			args{"key", "xoxd-412451%2Babcdef"},
			&http.Cookie{
				Name:    "key",
				Value:   "xoxd-412451%2Babcdef",
				Path:    defaultPath,
				Domain:  defaultDomain,
				Expires: timeFunc().AddDate(10, 0, 0),
				Secure:  true,
			},
		},
		{
			"URL Unsafe values are escaped",
			args{"key", "xoxd-412451+abcdef"},
			&http.Cookie{
				Name:    "key",
				Value:   "xoxd-412451%2Babcdef",
				Path:    defaultPath,
				Domain:  defaultDomain,
				Expires: timeFunc().AddDate(10, 0, 0),
				Secure:  true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, makeCookie(tt.args.key, tt.args.val))
			if got := makeCookie(tt.args.key, tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeCookie() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_urlsafe(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"safe", args{"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789%-._~"}, true},
		{"escaped", args{"a%2Bbc"}, true},
		{"unsafe 1", args{"ab+c"}, false},
		{"unsafe 2", args{"ab c"}, false},
		{"unsafe 3", args{"!abc"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := urlsafe(tt.args.s); got != tt.want {
				t.Errorf("urlsafe() = %v, want %v", got, tt.want)
			}
		})
	}
}
