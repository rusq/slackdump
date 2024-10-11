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
		w.Write(response)
	}))
}
