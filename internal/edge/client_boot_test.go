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

package edge

import (
	_ "embed"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed assets/client.userBoot.json
var clientUserBootJSON []byte

func TestClient_ClientUserBoot(t *testing.T) {
	srv := testServer(http.StatusOK, clientUserBootJSON)
	defer srv.Close()

	cl := Client{
		cl:           http.DefaultClient,
		edgeAPI:      srv.URL + "/",
		webclientAPI: srv.URL + "/",
	}
	r, err := cl.ClientUserBoot(t.Context())
	require.NoError(t, err)
	assert.True(t, r.Ok)
	assert.Equal(t, 3, len(r.Channels))
	assert.Equal(t, 4, len(r.IMs))
}
