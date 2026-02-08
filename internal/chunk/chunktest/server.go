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
