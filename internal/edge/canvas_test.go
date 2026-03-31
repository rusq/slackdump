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

//go:embed assets/quip.lookupThreadIds.json
var quipLookupThreadIdsJSON []byte

func TestClient_QuipLookupThreadIDs(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		srv := testServer(http.StatusOK, quipLookupThreadIdsJSON)
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			edgeAPI:      srv.URL + "/",
			webclientAPI: srv.URL + "/",
		}
		got, err := cl.QuipLookupThreadIDs(t.Context(), "F06R4HA3ZS8")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"F06R4HA3ZS8": "OYP9AAsR28Y"}, got)
	})
	t.Run("api error", func(t *testing.T) {
		srv := testServer(http.StatusOK, []byte(`{"ok": false, "error": "invalid_auth"}`))
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			edgeAPI:      srv.URL + "/",
			webclientAPI: srv.URL + "/",
		}
		_, err := cl.QuipLookupThreadIDs(t.Context(), "F06R4HA3ZS8")
		require.Error(t, err)
	})
	t.Run("500", func(t *testing.T) {
		srv := testServer(http.StatusInternalServerError, nil)
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			edgeAPI:      srv.URL + "/",
			webclientAPI: srv.URL + "/",
		}
		_, err := cl.QuipLookupThreadIDs(t.Context(), "F06R4HA3ZS8")
		require.Error(t, err)
	})
	t.Run("empty fileID slice", func(t *testing.T) {
		// no HTTP server needed — should return early
		cl := Client{}
		got, err := cl.QuipLookupThreadIDs(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
