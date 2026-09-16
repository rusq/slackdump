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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SavedList_paginates(t *testing.T) {
	var cursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/saved.list", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "saved", r.FormValue("filter"))
		require.Equal(t, "true", r.FormValue("include_tombstones"))
		cursors = append(cursors, r.FormValue("cursor"))
		if r.FormValue("cursor") == "" {
			_, _ = w.Write([]byte(`{
				"ok": true,
				"saved_items": [{"item_id":"C01","item_type":"message","ts":"1.000001","state":"in_progress","todo_state":"saved"}],
				"counts": {"total_count": 2},
				"response_metadata": {"next_cursor":"cursor-2"}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"ok": true,
			"saved_items": [{"item_id":"C02","item_type":"message","ts":"2.000002","state":"in_progress","todo_state":"to_do"}],
			"counts": {"total_count": 2},
			"response_metadata": {"next_cursor":""}
		}`))
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.SavedList(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"", "cursor-2"}, cursors)
	assert.Equal(t, "C01", got[0].ItemID)
	assert.Equal(t, "1.000001", got[0].Timestamp)
	assert.Equal(t, "saved", got[0].TodoState)
	assert.Equal(t, "C02", got[1].ItemID)
	assert.Equal(t, "to_do", got[1].TodoState)
}

func TestClient_SavedList_apiError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok": false, "error": "invalid_auth"}`))
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	_, err := cl.SavedList(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_auth")
}
