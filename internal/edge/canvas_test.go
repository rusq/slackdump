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
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

func TestCanvasChannelFromFileID(t *testing.T) {
	assert.Equal(t, "C06R4HA3ZS8", canvasChannelFromFileID("F06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID(""))
	assert.Equal(t, "", canvasChannelFromFileID("C06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID("X06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID("F"))
}

func Test_encodeEditor1Request(t *testing.T) {
	const oypID = "OYP9AAsR28Y"
	body := encodeEditor1Request(oypID)

	assert.Equal(t, oypID, string(pbGet(body, 1)))
	assert.Equal(t, oypID, string(pbGet(body, 3, 2)))
	assert.Equal(t, "editor", string(pbGet(body, 5)))
	kind, ok := pbGetVarint(pbGet(body, 3), 1)
	require.True(t, ok)
	assert.Equal(t, uint64(1), kind)
	version, ok := pbGetVarint(body, 6)
	require.True(t, ok)
	assert.Equal(t, uint64(1), version)
}

func appendCanvasThreadMetadata(table []byte, threadID string, replyCount, latestUS uint64) []byte {
	var entry []byte
	entry = protowire.AppendTag(entry, 1, protowire.BytesType)
	entry = protowire.AppendString(entry, threadID)
	entry = protowire.AppendTag(entry, 2, protowire.VarintType)
	entry = protowire.AppendVarint(entry, replyCount)
	entry = protowire.AppendTag(entry, 4, protowire.VarintType)
	entry = protowire.AppendVarint(entry, latestUS)
	table = protowire.AppendTag(table, 1, protowire.BytesType)
	return protowire.AppendBytes(table, entry)
}

func buildEditor1Response(entries ...canvasThreadMetadataFixture) []byte {
	var table []byte
	for _, entry := range entries {
		table = appendCanvasThreadMetadata(table, entry.threadID, entry.replyCount, entry.latestUS)
	}

	var field16 []byte
	field16 = protowire.AppendTag(field16, 16, protowire.BytesType)
	field16 = protowire.AppendBytes(field16, table)

	var field6 []byte
	field6 = protowire.AppendTag(field6, 6, protowire.BytesType)
	field6 = protowire.AppendBytes(field6, field16)

	var inner []byte
	inner = protowire.AppendTag(inner, 2, protowire.BytesType)
	inner = protowire.AppendBytes(inner, field6)

	var body []byte
	body = protowire.AppendTag(body, 2, protowire.BytesType)
	return protowire.AppendBytes(body, inner)
}

type canvasThreadMetadataFixture struct {
	threadID   string
	replyCount uint64
	latestUS   uint64
}

func Test_decodeCanvasThreadMetadata(t *testing.T) {
	t.Run("mixed discussions", func(t *testing.T) {
		body := buildEditor1Response(
			canvasThreadMetadataFixture{
				threadID:   "temp:C:OYPefc4c7420fb142be9ed33e878",
				replyCount: 2,
				latestUS:   1773451290951589,
			},
			canvasThreadMetadataFixture{
				threadID:   "temp:C:OYP4a8f0bc7b9b747c7abaf7bf1e",
				replyCount: 0,
				latestUS:   1773451370010299,
			},
		)

		got, err := decodeCanvasThreadMetadata(body)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, canvasThreadMetadata{
			ThreadID:        "temp:C:OYPefc4c7420fb142be9ed33e878",
			ReplyCount:      2,
			LatestMessageTS: "1773451290.951589",
		}, got[0])
		assert.Equal(t, "1773451370.010299", got[1].LatestMessageTS)
		assert.Zero(t, got[1].ReplyCount)
	})

	t.Run("empty table", func(t *testing.T) {
		got, err := decodeCanvasThreadMetadata(buildEditor1Response())
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("missing table", func(t *testing.T) {
		_, err := decodeCanvasThreadMetadata(nil)
		require.ErrorIs(t, err, errCanvasMissingThreadTable)
	})

	t.Run("missing thread ID", func(t *testing.T) {
		_, err := decodeCanvasThreadMetadata(buildEditor1Response(canvasThreadMetadataFixture{
			latestUS: 1773451370010299,
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing thread ID")
	})

	t.Run("missing timestamp", func(t *testing.T) {
		_, err := decodeCanvasThreadMetadata(buildEditor1Response(canvasThreadMetadataFixture{
			threadID: "temp:C:thread",
		}))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing latest message timestamp")
	})
}

func Test_resolvedCanvasRootTS(t *testing.T) {
	tests := []struct {
		name     string
		metadata canvasThreadMetadata
		probe    []canvasAPIMessage
		want     string
		wantErr  bool
	}{
		{
			name: "zero replies uses metadata timestamp",
			metadata: canvasThreadMetadata{
				ThreadID:        "temp:C:zero",
				LatestMessageTS: "1773451370.010299",
			},
			want: "1773451370.010299",
		},
		{
			name: "reply resolves parent timestamp",
			metadata: canvasThreadMetadata{
				ThreadID:        "temp:C:replied",
				ReplyCount:      2,
				LatestMessageTS: "1773451290.951589",
			},
			probe: []canvasAPIMessage{{
				TS:       "1773451290.951589",
				ThreadTS: "1773451284.332529",
			}},
			want: "1773451284.332529",
		},
		{
			name: "missing matching reply",
			metadata: canvasThreadMetadata{
				ThreadID:        "temp:C:replied",
				ReplyCount:      1,
				LatestMessageTS: "1773451290.951589",
			},
			probe:   []canvasAPIMessage{{TS: "1773451284.332529"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvedCanvasRootTS(tt.metadata, tt.probe)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_canvasRootMessage(t *testing.T) {
	const (
		threadID = "temp:C:OYPefc4c7420fb142be9ed33e878"
		rootTS   = "1773451284.332529"
	)
	metadata := canvasThreadMetadata{ThreadID: threadID}

	t.Run("normalises root thread timestamp", func(t *testing.T) {
		got, err := canvasRootMessage(metadata, rootTS, []canvasAPIMessage{{
			TS:         rootTS,
			SubType:    "document_comment_root",
			Text:       "Check list",
			ReplyCount: 2,
			DocumentComment: CanvasDocumentComment{
				ThreadID: threadID,
				Authors:  []string{"UHSD97ZA5"},
			},
		}})
		require.NoError(t, err)
		assert.Equal(t, rootTS, got.TS)
		assert.Equal(t, rootTS, got.ThreadTS)
		assert.Equal(t, "Check list", got.Text)
		assert.Equal(t, 2, got.ReplyCount)
		assert.Equal(t, threadID, got.DocumentComment.ThreadID)
	})

	t.Run("rejects mismatched thread", func(t *testing.T) {
		_, err := canvasRootMessage(metadata, rootTS, []canvasAPIMessage{{
			TS:      rootTS,
			SubType: "document_comment_root",
			DocumentComment: CanvasDocumentComment{
				ThreadID: "temp:C:different",
			},
		}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "thread ID mismatch")
	})
}

func TestClient_CanvasThreadRoots(t *testing.T) {
	const (
		fileID        = "F06R4HA3ZS8"
		channelID     = "C06R4HA3ZS8"
		oypID         = "OYP9AAsR28Y"
		repliedThread = "temp:C:OYPefc4c7420fb142be9ed33e878"
		zeroThread    = "temp:C:OYP4a8f0bc7b9b747c7abaf7bf1e"
		replyTS       = "1773451290.951589"
		repliedRootTS = "1773451284.332529"
		zeroRootTS    = "1773451370.010299"
	)

	t.Run("discovers and resolves current roots", func(t *testing.T) {
		editorBody := buildEditor1Response(
			canvasThreadMetadataFixture{
				threadID:   repliedThread,
				replyCount: 2,
				latestUS:   1773451290951589,
			},
			canvasThreadMetadataFixture{
				threadID:   zeroThread,
				replyCount: 0,
				latestUS:   1773451370010299,
			},
			canvasThreadMetadataFixture{
				threadID:   zeroThread,
				replyCount: 0,
				latestUS:   1773451370010299,
			},
		)

		var replyTimestamps []string
		var editorSessionID string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			switch r.URL.Path {
			case "/api/quip.lookupThreadIds":
				assert.Equal(t, fileID, r.FormValue("file_ids"))
				_, _ = io.WriteString(w, `{"ok":true,"lookup":{"`+fileID+`":"`+oypID+`"}}`)
			case "/canvas/collab/controller-init":
				assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
				_, _ = io.WriteString(w, `{"user_id":"ZEH9EA5279K"}`)
			case "/canvas/-/load-data/editor/1":
				assert.Equal(t, "ZEH9EA5279K", r.FormValue("_user_id"))
				assert.Equal(t, "10", r.FormValue("_version"))
				editorSessionID = r.FormValue("_window_session_id")
				assert.Len(t, editorSessionID, 11)
				requestBody, err := base64.StdEncoding.DecodeString(r.FormValue("request_binary"))
				require.NoError(t, err)
				assert.Equal(t, oypID, string(pbGet(requestBody, 1)))
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(editorBody)
			case "/api/conversations.replies":
				assert.Equal(t, channelID, r.FormValue("channel"))
				assert.Equal(t, "1000", r.FormValue("limit"))
				ts := r.FormValue("ts")
				replyTimestamps = append(replyTimestamps, ts)
				switch ts {
				case replyTS:
					_, _ = io.WriteString(w, fmt.Sprintf(
						`{"ok":true,"messages":[{"ts":%q,"thread_ts":%q}]}`,
						replyTS,
						repliedRootTS,
					))
				case repliedRootTS:
					_, _ = io.WriteString(w, fmt.Sprintf(
						`{"ok":true,"messages":[{"ts":%q,"subtype":"document_comment_root","text":"Check list","reply_count":2,"document_comment":{"thread_id":%q,"authors":["UHSD97ZA5"]}}]}`,
						repliedRootTS,
						repliedThread,
					))
				case zeroRootTS:
					_, _ = io.WriteString(w, fmt.Sprintf(
						`{"ok":true,"messages":[{"ts":%q,"subtype":"document_comment_root","text":"Zero replies","reply_count":0,"document_comment":{"thread_id":%q}}]}`,
						zeroRootTS,
						zeroThread,
					))
				default:
					t.Fatalf("unexpected conversations.replies timestamp %q", ts)
				}
			default:
				t.Fatalf("unexpected request path %q", r.URL.Path)
			}
		}))
		defer srv.Close()

		cl := Client{
			cl:           http.DefaultClient,
			webclientAPI: srv.URL + "/api/",
			token:        "xoxc-test",
		}
		got, err := cl.CanvasThreadRoots(t.Context(), fileID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, []string{replyTS, repliedRootTS, zeroRootTS}, replyTimestamps)
		assert.NotEmpty(t, editorSessionID)
		assert.Equal(t, repliedRootTS, got[0].TS)
		assert.Equal(t, repliedRootTS, got[0].ThreadTS)
		assert.Equal(t, "Check list", got[0].Text)
		assert.Equal(t, repliedThread, got[0].DocumentComment.ThreadID)
		assert.Equal(t, zeroRootTS, got[1].TS)
		assert.Equal(t, zeroRootTS, got[1].ThreadTS)
	})

	t.Run("invalid file ID", func(t *testing.T) {
		cl := Client{}
		_, err := cl.CanvasThreadRoots(t.Context(), channelID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid file ID")
	})

	t.Run("missing document mapping", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"ok":true,"lookup":{}}`)
		}))
		defer srv.Close()

		cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
		_, err := cl.CanvasThreadRoots(t.Context(), fileID)
		require.ErrorIs(t, err, errCanvasMissingOYPID)
	})

	t.Run("missing controller user ID", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "quip.lookupThreadIds") {
				_, _ = io.WriteString(w, `{"ok":true,"lookup":{"`+fileID+`":"`+oypID+`"}}`)
				return
			}
			_, _ = io.WriteString(w, `{}`)
		}))
		defer srv.Close()

		cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
		_, err := cl.CanvasThreadRoots(t.Context(), fileID)
		require.ErrorIs(t, err, errCanvasMissingUserID)
	})
}
