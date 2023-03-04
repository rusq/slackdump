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
