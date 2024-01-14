// Package chunktest provides a test server for testing the chunk package.
package chunktest

import (
	"io"
	"log"
	"net/http/httptest"
	"os"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

var lg = log.New(os.Stderr, "chunktest: ", log.LstdFlags)

// Server is a test server for testing the chunk package, that serves API
// from a single chunk file.
type Server struct {
	baseServer
	p *chunk.Player
}

// NewServer returns a new Server, it requires the chunk file handle in rs, and
// an ID of the user that will be returned by AuthTest in currentUserID.
func NewServer(rs io.ReadSeeker, currentUserID string) *Server {
	p, err := chunk.NewPlayer(rs)
	if err != nil {
		panic(err)
	}
	return &Server{
		baseServer: baseServer{Server: httptest.NewServer(router(p, currentUserID))},
		p:          p,
	}
}
