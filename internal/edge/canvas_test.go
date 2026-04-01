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

func TestCanvasChannelFromFileID(t *testing.T) {
	assert.Equal(t, "C06R4HA3ZS8", canvasChannelFromFileID("F06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID(""))
	assert.Equal(t, "", canvasChannelFromFileID("C06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID("X06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID("F"))
}

func TestClient_conversationsHistoryForCanvas_paginates(t *testing.T) {
	var cursors []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/conversations.history", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "C06R4HA3ZS8", r.FormValue("channel"))
		require.Equal(t, "1000", r.FormValue("limit"))
		cursors = append(cursors, r.FormValue("cursor"))
		if r.FormValue("cursor") == "" {
			_, _ = w.Write([]byte(`{
				"ok": true,
				"messages": [{"ts":"1.000001","subtype":"document_comment_root","text":"one","document_comment":{"thread_id":"temp:C:one"}}],
				"response_metadata": {"next_cursor":"cursor-2"}
			}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"ok": true,
			"messages": [{"ts":"2.000002","subtype":"document_comment_root","text":"two","document_comment":{"thread_id":"temp:C:two"}}],
			"response_metadata": {"next_cursor":""}
		}`))
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.conversationsHistoryForCanvas(t.Context(), "C06R4HA3ZS8")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, []string{"", "cursor-2"}, cursors)
	assert.Equal(t, "1.000001", got[0].TS)
	assert.Equal(t, "2.000002", got[1].TS)
}

func TestClient_CanvasThreadRoots(t *testing.T) {
	var seenPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "/conversations.history", r.URL.Path)
		require.Equal(t, "C06R4HA3ZS8", r.FormValue("channel"))
		require.Equal(t, "1000", r.FormValue("limit"))
		_, _ = w.Write([]byte(`{
			"ok": true,
			"messages": [
				{
					"ts": "1773451284.332529",
					"subtype": "document_comment_root",
					"text": "Check list",
					"reply_count": 2,
					"document_comment": {"thread_id": "temp:C:OYPefc4c7420fb142be9ed33e878"}
				},
				{
					"ts": "1773451400.057089",
					"thread_ts": "1773451400.057089",
					"subtype": "document_comment_root",
					"text": "heading",
					"reply_count": 1,
					"document_comment": {"thread_id": "temp:C:OYPe240dbecfbc449eaa962b60d8"}
				},
				{
					"ts": "1773451405.000000",
					"subtype": "message",
					"text": "ignore me"
				}
			]
		}`))
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.CanvasThreadRoots(t.Context(), "F06R4HA3ZS8")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1773451284.332529", got[0].TS)
	assert.Equal(t, "1773451284.332529", got[0].ThreadTS)
	assert.Equal(t, "1773451400.057089", got[1].ThreadTS)
	assert.Equal(t, "temp:C:OYPefc4c7420fb142be9ed33e878", got[0].DocumentComment.ThreadID)
	assert.Equal(t, []string{"/conversations.history"}, seenPaths)
}

func TestClient_CanvasThreadRoots_invalidFileID(t *testing.T) {
	cl := Client{}
	_, err := cl.CanvasThreadRoots(t.Context(), "C06R4HA3ZS8")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file ID")
}
