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
package fixtures

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed assets/slack_api/authtestinfo.json
var TestAuthTestInfo []byte

// TestAuthServer returns a test HTTP server that responds with the test auth
// info [TestAuthTestInfo].  The caller should close the server when done.
func TestAuthServer(t *testing.T) *httptest.Server {
	t.Helper()
	return TestServer(t, http.StatusOK, TestAuthTestInfo)
}

// TestServer returns a test HTTP server that responds with the given code and
// response. The caller should close the server when done.
func TestServer(t *testing.T, code int, response []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write(response)
	}))
}
