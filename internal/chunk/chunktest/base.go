package chunktest

import "net/http/httptest"

// baseServer is a wrapper arund the test HTTP server with some overrides.
type baseServer struct {
	*httptest.Server
}

// URL returns the server URL.
func (s *baseServer) URL() string {
	return s.Server.URL + "/api/"
}
