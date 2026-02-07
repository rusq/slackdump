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
	"bytes"
	"context"
	_ "embed"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_getTokenByCookie(t *testing.T) {
	oldTimeFunc := timeFunc
	timeFunc = func() time.Time {
		return time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		timeFunc = oldTimeFunc
	})

	type args struct {
		ctx           context.Context
		workspaceName string
		dCookie       string
	}
	tests := []struct {
		name     string
		args     args
		testBody []byte
		want     string
		want1    []*http.Cookie
		wantErr  bool
	}{
		{
			name: "finds the token and cookies",
			args: args{
				ctx:           t.Context(),
				workspaceName: "test",
				dCookie:       "dcookie",
			},
			testBody: testBody,
			want:     "xoxc-000000000300-604451271345-8802919159412-ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			want1: []*http.Cookie{
				{Name: "unit", Value: "test", Raw: "unit=test"},
				makeCookie("d", "dcookie"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.SetCookie(w, &http.Cookie{Name: "unit", Value: "test"})
				io.Copy(w, bytes.NewReader(tt.testBody))
			}))
			ssbURI = func(string) string {
				return srv.URL
			}
			got, got1, err := getTokenByCookie(tt.args.ctx, tt.args.workspaceName, tt.args.dCookie)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTokenByCookie() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getTokenByCookie() got = %v, want %v", got, tt.want)
			}
			assert.EqualExportedValues(t, tt.want1, got1)
		})
	}
}

//go:embed testdata/redirect.html
var testBody []byte

func Test_extractToken(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "extracts token from the HTML body",
			args:    args{r: bytes.NewReader(testBody)},
			want:    "xoxc-000000000300-604451271345-8802919159412-ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			wantErr: false,
		},
		{
			name:    "no token is an error",
			args:    args{strings.NewReader("first line\nsecond line\n")},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractToken(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
