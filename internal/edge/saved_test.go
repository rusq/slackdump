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

func TestClient_SavedList_queriesAllFilters(t *testing.T) {
	var filters []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/saved.list", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "true", r.FormValue("include_tombstones"))
		filter := r.FormValue("filter")

		switch {
		case filter == "saved" && r.FormValue("cursor") == "":
			filters = append(filters, filter)
			_, _ = w.Write([]byte(`{
				"ok": true,
				"saved_items": [{"item_id":"C01","item_type":"message","ts":"1.000001","state":"in_progress","todo_state":"saved"}],
				"response_metadata": {"next_cursor":"cursor-2"}
			}`))
		case filter == "saved" && r.FormValue("cursor") == "cursor-2":
			_, _ = w.Write([]byte(`{
				"ok": true,
				"saved_items": [{"item_id":"C02","item_type":"message","ts":"2.000002","state":"in_progress","todo_state":"saved"}],
				"response_metadata": {"next_cursor":""}
			}`))
		case filter == "completed":
			filters = append(filters, filter)
			_, _ = w.Write([]byte(`{
				"ok": true,
				"saved_items": [{"item_id":"C03","item_type":"message","ts":"3.000003","state":"completed","todo_state":"completed"}],
				"response_metadata": {"next_cursor":""}
			}`))
		case filter == "archived":
			filters = append(filters, filter)
			_, _ = w.Write([]byte(`{
				"ok": true,
				"saved_items": [],
				"response_metadata": {"next_cursor":""}
			}`))
		default:
			t.Fatalf("unexpected filter/cursor: filter=%q cursor=%q", filter, r.FormValue("cursor"))
		}
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.SavedList(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.ElementsMatch(t, []string{"saved", "completed", "archived"}, filters)
	assert.ElementsMatch(t, []string{"C01", "C02", "C03"}, []string{got[0].ItemID, got[1].ItemID, got[2].ItemID})
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
