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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

//go:embed assets/quip.lookupThreadIds.json
var quipLookupThreadIdsJSON []byte

//go:embed assets/messages.list.json
var messagesListJSON []byte

func Test_encodeEditor1Request(t *testing.T) {
	got := encodeEditor1Request("OYP9AAsR28Y")
	want, _ := base64.StdEncoding.DecodeString(
		"CgtPWVA5QUFzUjI4WRoPCAESC09ZUDlBQXNSMjhZKgZlZGl0b3IwAQ==")
	assert.Equal(t, want, got)
}

func Test_encodeEditor2Request(t *testing.T) {
	got := encodeEditor2Request("Cca9cA1qpvy", "OYP9iAsR28Y")
	want, _ := base64.StdEncoding.DecodeString(
		"CgtDY2E5Y0ExcXB2eQoLT1lQOWlBc1IyOFkqBmVkaXRvcjAC")
	assert.Equal(t, want, got)
}

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

func Test_pbGet(t *testing.T) {
	// outer: f1(bytes)="hello", f3(bytes) containing f7(bytes)="world"
	inner := protowire.AppendTag(nil, 7, protowire.BytesType)
	inner = protowire.AppendString(inner, "world")

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, "hello")
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)

	assert.Equal(t, []byte("hello"), pbGet(b, 1), "single field")
	assert.Equal(t, []byte("world"), pbGet(b, 3, 7), "nested field")
	assert.Nil(t, pbGet(b, 99), "missing field")
	assert.Nil(t, pbGet(b, 3, 99), "missing nested field")
	assert.Nil(t, pbGet(nil), "nil input")
}

func Test_pbGetAll(t *testing.T) {
	// f1(bytes)="a", f1(bytes)="b", f2(bytes) containing f1(bytes)="c" + f1(bytes)="d"
	inner := protowire.AppendTag(nil, 1, protowire.BytesType)
	inner = protowire.AppendString(inner, "c")
	inner = protowire.AppendTag(inner, 1, protowire.BytesType)
	inner = protowire.AppendString(inner, "d")

	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, "a")
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, "b")
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)

	assert.Equal(t, [][]byte{[]byte("a"), []byte("b")}, pbGetAll(b, 1), "top-level repeated")
	assert.Equal(t, [][]byte{[]byte("c"), []byte("d")}, pbGetAll(b, 2, 1), "nested repeated")
	assert.Nil(t, pbGetAll(b), "empty path")
	assert.Nil(t, pbGetAll(b, 99, 1), "missing parent")
}

func Test_pbFirstString(t *testing.T) {
	// varint first, then string
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, 42)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, "hello")
	assert.Equal(t, "hello", pbFirstString(b), "string after varint")

	// string first
	var b2 []byte
	b2 = protowire.AppendTag(b2, 1, protowire.BytesType)
	b2 = protowire.AppendString(b2, "world")
	assert.Equal(t, "world", pbFirstString(b2), "string first")

	// only varints
	var b3 []byte
	b3 = protowire.AppendTag(b3, 1, protowire.VarintType)
	b3 = protowire.AppendVarint(b3, 1)
	assert.Equal(t, "", pbFirstString(b3), "no string fields")

	assert.Equal(t, "", pbFirstString(nil), "nil input")
}

func Test_decodeControllerInitOptions(t *testing.T) {
	opts := base64.StdEncoding.EncodeToString(encodeControllerInitOptions("Cca9cA1qpvy", "THY5HTZ8U"))
	got, err := decodeControllerInitOptions(opts)
	require.NoError(t, err)
	assert.Equal(t, "Cca9cA1qpvy", got)
}

func Test_decodeControllerInitOptions_missingSession(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 52, protowire.BytesType)
	b = protowire.AppendString(b, "Cca9cA1qpvy")

	_, err := decodeControllerInitOptions(base64.StdEncoding.EncodeToString(b))
	require.ErrorIs(t, err, errCanvasMissingSession)
}

func Test_decodeControllerInitOptions_mismatch(t *testing.T) {
	var session []byte
	session = protowire.AppendTag(session, 1, protowire.BytesType)
	session = protowire.AppendString(session, "Cca9cA1qpvy")

	var b []byte
	b = protowire.AppendTag(b, 43, protowire.BytesType)
	b = protowire.AppendBytes(b, session)
	b = protowire.AppendTag(b, 52, protowire.BytesType)
	b = protowire.AppendString(b, "otherSession")

	_, err := decodeControllerInitOptions(base64.StdEncoding.EncodeToString(b))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session mismatch")
}

// buildEditor1Response constructs a minimal load-data/editor/1 protobuf response.
// The actual API response wraps the document state in body.f2.f2, so we mirror
// that structure here: content lives inside two nested field-2 bytes fields.
// Each f124 entry uses:
//
//	f1: root message timestamp in microseconds
//	f2[]: optional block IDs attached to that thread root
func buildEditor1Response(section2OYP string, timestamps []string) []byte {
	var f3 []byte
	for _, ts := range timestamps {
		secs, micros, ok := strings.Cut(ts, ".")
		if !ok {
			panic("invalid Slack timestamp in test fixture: " + ts)
		}
		secVal, err := strconv.ParseUint(secs, 10, 64)
		if err != nil {
			panic(err)
		}
		microVal, err := strconv.ParseUint(micros, 10, 64)
		if err != nil {
			panic(err)
		}
		var entry []byte
		entry = protowire.AppendTag(entry, 1, protowire.VarintType)
		entry = protowire.AppendVarint(entry, secVal*1_000_000+microVal)
		f3 = protowire.AppendTag(f3, 124, protowire.BytesType)
		f3 = protowire.AppendBytes(f3, entry)
	}
	var f7 []byte
	for i, ts := range timestamps {
		blockID := fmt.Sprintf("temp:C:OYPblock%d", i)
		threadID := fmt.Sprintf("temp:C:OYPthread%d", i)
		secs, micros, _ := strings.Cut(ts, ".")
		secVal, _ := strconv.ParseUint(secs, 10, 64)
		microVal, _ := strconv.ParseUint(micros, 10, 64)
		sectionCreated := secVal*1_000_000 + microVal
		sectionEdited := sectionCreated + 1234

		var entry []byte
		entry = protowire.AppendTag(entry, 1, protowire.BytesType)
		entry = protowire.AppendString(entry, blockID)
		entry = protowire.AppendTag(entry, 6, protowire.BytesType)
		entry = protowire.AppendString(entry, "OYP9AAsR28Y")
		entry = protowire.AppendTag(entry, 7, protowire.BytesType)
		entry = protowire.AppendString(entry, "OYP9BAL4BMO")
		entry = protowire.AppendTag(entry, 12, protowire.BytesType)
		entry = protowire.AppendString(entry, `<annotation id="`+threadID+`">text</annotation>`)
		entry = protowire.AppendTag(entry, 26, protowire.VarintType)
		entry = protowire.AppendVarint(entry, sectionCreated)
		entry = protowire.AppendTag(entry, 27, protowire.VarintType)
		entry = protowire.AppendVarint(entry, sectionEdited)
		entry = protowire.AppendTag(entry, 33, protowire.BytesType)
		entry = protowire.AppendString(entry, fmt.Sprintf("record-%d", i))
		f7 = protowire.AppendTag(f7, 7, protowire.BytesType)
		f7 = protowire.AppendBytes(f7, entry)
	}
	var f6 []byte
	f6 = protowire.AppendTag(f6, 63, protowire.BytesType)
	f6 = protowire.AppendString(f6, section2OYP)

	// inner f2 (body.f2.f2) — the document state
	var inner []byte
	inner = protowire.AppendTag(inner, 3, protowire.BytesType)
	inner = protowire.AppendBytes(inner, f3)
	inner = protowire.AppendTag(inner, 6, protowire.BytesType)
	inner = protowire.AppendBytes(inner, f6)
	inner = append(inner, f7...)

	// outer f2 (body.f2)
	var outerF2 []byte
	outerF2 = protowire.AppendTag(outerF2, 2, protowire.BytesType)
	outerF2 = protowire.AppendBytes(outerF2, inner)

	var b []byte
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, outerF2)
	return b
}

// buildEditor2Response constructs a minimal load-data/editor/2 protobuf response.
func buildEditor2Response(blockIDs []string) []byte {
	var f55 []byte
	for _, id := range blockIDs {
		var f8 []byte
		f8 = protowire.AppendTag(f8, 3, protowire.BytesType)
		f8 = protowire.AppendString(f8, id)
		f55 = protowire.AppendTag(f55, 8, protowire.BytesType)
		f55 = protowire.AppendBytes(f55, f8)
	}
	var innerF2 []byte
	innerF2 = protowire.AppendTag(innerF2, 55, protowire.BytesType)
	innerF2 = protowire.AppendBytes(innerF2, f55)

	var outerF2 []byte
	outerF2 = protowire.AppendTag(outerF2, 2, protowire.BytesType)
	outerF2 = protowire.AppendBytes(outerF2, innerF2)

	var b []byte
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendBytes(b, outerF2)
	return b
}

func TestClient_stepEditor1(t *testing.T) {
	payload := buildEditor1Response("OYP9iAsR28Y", []string{"1773451284.332529", "1773451370.010299"})
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	got, err := cl.stepEditor1(t.Context(), "OYP9AAsR28Y", "testSession", "UTEST123")
	require.NoError(t, err)
	assert.Equal(t, "OYP9iAsR28Y", got.Section2OYP)
	assert.Equal(t, []string{"1773451284.332529", "1773451370.010299"}, got.CandidateTimestamps)
	require.Len(t, got.ThreadRecords, 2)
	assert.Equal(t, "temp:C:OYPblock0", got.ThreadRecords[0].BlockID)
	assert.Equal(t, "temp:C:OYPthread0", got.ThreadRecords[0].ThreadID)
	assert.Equal(t, "1773451284.332529", got.ThreadRecords[0].SectionCreatedTS)
}

func TestClient_stepEditor1_missingSection2(t *testing.T) {
	payload := buildEditor1Response("", []string{"1773451284.332529"})
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	_, err := cl.stepEditor1(t.Context(), "OYP9AAsR28Y", "testSession", "UTEST123")
	require.ErrorIs(t, err, errCanvasMissingSection2OYP)
}

func TestClient_stepEditor1_missingTimestamps(t *testing.T) {
	payload := buildEditor1Response("OYP9iAsR28Y", nil)
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	_, err := cl.stepEditor1(t.Context(), "OYP9AAsR28Y", "testSession", "UTEST123")
	require.ErrorIs(t, err, errCanvasMissingThreadTS)
}

func TestClient_stepEditor2(t *testing.T) {
	blockIDs := []string{
		"temp:C:OYPefc4c7420fb142be9ed33e878",
		"temp:C:OYP1f946863c7b145229104df82f",
	}
	payload := buildEditor2Response(blockIDs)
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	got, err := cl.stepEditor2(t.Context(), "testSession", "OYP9iAsR28Y")
	require.NoError(t, err)
	assert.Equal(t, blockIDs, got)
}

func TestClient_stepEditor2_noReplies(t *testing.T) {
	// empty f2.f2.f55 — no blocks with replies
	payload := buildEditor2Response(nil)
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	got, err := cl.stepEditor2(t.Context(), "testSession", "OYP9iAsR28Y")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestClient_stepEditor2_malformed(t *testing.T) {
	srv := testServer(http.StatusOK, []byte("not-protobuf"))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	_, err := cl.stepEditor2(t.Context(), "testSession", "OYP9iAsR28Y")
	require.ErrorIs(t, err, errCanvasMalformedEditor2)
}

func TestClient_stepControllerInit(t *testing.T) {
	payload, err := json.Marshal(canvasControllerInitResponse{
		InitOptions: base64.StdEncoding.EncodeToString(encodeControllerInitOptions("Cca9cA1qpvy", "THY5HTZ8U")),
		UserID:      "UTEST123",
	})
	require.NoError(t, err)

	var seenContentType string
	var seenBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/", token: "xoxc-test"}
	got, err := cl.stepControllerInit(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "UTEST123", got.UserID)
	// SessionID is no longer extracted from controller-init; it is generated client-side.
	assert.Empty(t, got.SessionID)
	assert.Contains(t, seenContentType, "multipart/form-data")
	assert.Contains(t, seenBody, `name="token"`)
	assert.Contains(t, seenBody, "xoxc-test")
}

func TestClient_stepMessagesList(t *testing.T) {
	srv := testServer(http.StatusOK, messagesListJSON)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.stepMessagesList(t.Context(), "C06R4HA3ZS8", []string{"1773451284.332529", "1773451370.010299"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1773451284.332529", got[0].TS)
	assert.Equal(t, "Check list", got[0].Text)
	assert.Equal(t, 2, got[0].ReplyCount)
	assert.Equal(t, "temp:C:OYPefc4c7420fb142be9ed33e878", got[0].DocumentComment.ThreadID)
}

func TestClient_stepMessagesList_objectShape(t *testing.T) {
	payload := []byte(`{
		"ok": true,
		"messages": {
			"C06R4HA3ZS8": {
				"1773451370.010299": {
					"ts": "1773451370.010299",
					"thread_ts": "1773451370.010299",
					"text": "Another comment",
					"reply_count": 1,
					"document_comment": {
						"thread_id": "temp:C:OYP1f946863c7b145229104df82f",
						"authors": ["UHSD97ZA5"]
					}
				},
				"1773451284.332529": {
					"ts": "1773451284.332529",
					"thread_ts": "1773451284.332529",
					"text": "Check list",
					"reply_count": 2,
					"document_comment": {
						"thread_id": "temp:C:OYPefc4c7420fb142be9ed33e878",
						"authors": ["UHSD97ZA5"]
					}
				}
			}
		}
	}`)
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.stepMessagesList(t.Context(), "C06R4HA3ZS8", []string{"1773451284.332529", "1773451370.010299"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1773451284.332529", got[0].TS)
	assert.Equal(t, "1773451370.010299", got[1].TS)
}

func TestClient_stepMessagesList_messagesDataShape(t *testing.T) {
	payload := []byte(`{
		"ok": true,
		"messages": {},
		"messages_data": {
			"C06R4HA3ZS8": {
				"messages": [
					{
						"ts": "1773451370.010299",
						"thread_ts": "1773451370.010299",
						"text": "Another comment",
						"reply_count": 1,
						"document_comment": {
							"thread_id": "temp:C:OYP1f946863c7b145229104df82f",
							"authors": ["UHSD97ZA5"]
						}
					},
					{
						"ts": "1773451284.332529",
						"thread_ts": "1773451284.332529",
						"text": "Check list",
						"reply_count": 2,
						"document_comment": {
							"thread_id": "temp:C:OYPefc4c7420fb142be9ed33e878",
							"authors": ["UHSD97ZA5"]
						}
					}
				]
			}
		}
	}`)
	srv := testServer(http.StatusOK, payload)
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	got, err := cl.stepMessagesList(t.Context(), "C06R4HA3ZS8", []string{"1773451284.332529", "1773451370.010299"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1773451284.332529", got[0].TS)
	assert.Equal(t, "1773451370.010299", got[1].TS)
}

func TestClient_stepMessagesList_apiError(t *testing.T) {
	srv := testServer(http.StatusOK, []byte(`{"ok": false, "error": "channel_not_found"}`))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	_, err := cl.stepMessagesList(t.Context(), "CINVALID", []string{"1773451284.332529"})
	require.Error(t, err)
}

func TestClient_CanvasThreadRoots(t *testing.T) {
	var seenPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "/conversations.history", r.URL.Path)
		require.Equal(t, "C06R4HA3ZS8", r.FormValue("channel"))
		require.Equal(t, "1000", r.FormValue("limit"))
		w.WriteHeader(http.StatusOK)
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
	assert.Equal(t, []string{"/conversations.history"}, seenPaths)
}

func TestCanvasChannelFromFileID(t *testing.T) {
	assert.Equal(t, "C06R4HA3ZS8", canvasChannelFromFileID("F06R4HA3ZS8"))
	assert.Equal(t, "", canvasChannelFromFileID(""))
	assert.Equal(t, "", canvasChannelFromFileID("C06R4HA3ZS8"))
}
