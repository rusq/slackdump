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
	"io"
	"net/http"
	"net/http/httptest"
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
func buildEditor1Response(section2OYP string, timestamps []string) []byte {
	var f3 []byte
	for _, ts := range timestamps {
		var entry []byte
		entry = protowire.AppendTag(entry, 1, protowire.BytesType)
		entry = protowire.AppendString(entry, ts)
		f3 = protowire.AppendTag(f3, 124, protowire.BytesType)
		f3 = protowire.AppendBytes(f3, entry)
	}
	var f6 []byte
	f6 = protowire.AppendTag(f6, 63, protowire.BytesType)
	f6 = protowire.AppendString(f6, section2OYP)

	var b []byte
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, f3)
	b = protowire.AppendTag(b, 6, protowire.BytesType)
	b = protowire.AppendBytes(b, f6)
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
	assert.Equal(t, []string{"1773451284.332529", "1773451370.010299"}, got.ThreadTimestamps)
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
	assert.Equal(t, "Cca9cA1qpvy", got.SessionID)
	assert.Equal(t, "UTEST123", got.UserID)
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

func TestClient_stepMessagesList_apiError(t *testing.T) {
	srv := testServer(http.StatusOK, []byte(`{"ok": false, "error": "channel_not_found"}`))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/"}
	_, err := cl.stepMessagesList(t.Context(), "CINVALID", []string{"1773451284.332529"})
	require.Error(t, err)
}

func TestClient_CanvasThreadRoots(t *testing.T) {
	controllerPayload, err := json.Marshal(canvasControllerInitResponse{
		InitOptions: base64.StdEncoding.EncodeToString(encodeControllerInitOptions("Cca9cA1qpvy", "THY5HTZ8U")),
		UserID:      "UTEST123",
	})
	require.NoError(t, err)
	e1payload := buildEditor1Response("OYP9iAsR28Y", []string{"1773451284.332529"})
	e2payload := buildEditor2Response([]string{"temp:C:OYPefc4c7420fb142be9ed33e878"})

	var seenPaths []string
	var editor1Body string
	var editor2Body string
	// Routing server: editor/1 and editor/2 return protobuf; messages.list returns JSON.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.URL.Path == "/canvas/collab/controller-init":
			w.WriteHeader(http.StatusOK)
			w.Write(controllerPayload)
		case r.URL.Path == "/canvas/-/load-data/editor/1":
			editor1Body = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write(e1payload)
		case r.URL.Path == "/canvas/-/load-data/editor/2":
			editor2Body = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write(e2payload)
		default:
			w.WriteHeader(http.StatusOK)
			w.Write(messagesListJSON)
		}
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	got, err := cl.CanvasThreadRoots(t.Context(), "OYP9AAsR28Y", "C06R4HA3ZS8", "UTEST123")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "1773451284.332529", got[0].TS)
	assert.Equal(t, []string{
		"/canvas/collab/controller-init",
		"/canvas/-/load-data/editor/1",
		"/canvas/-/load-data/editor/2",
		"/api/messages.list",
	}, seenPaths)
	assert.Contains(t, editor1Body, "_window_session_id=Cca9cA1qpvy")
	assert.Contains(t, editor2Body, "_window_session_id=Cca9cA1qpvy")
}

func TestClient_CanvasThreadRoots_userIDMismatch(t *testing.T) {
	controllerPayload, err := json.Marshal(canvasControllerInitResponse{
		InitOptions: base64.StdEncoding.EncodeToString(encodeControllerInitOptions("Cca9cA1qpvy", "THY5HTZ8U")),
		UserID:      "UOTHER",
	})
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/canvas/collab/controller-init" {
			w.WriteHeader(http.StatusOK)
			w.Write(controllerPayload)
			return
		}
		t.Fatalf("unexpected request path: %s", r.URL.Path)
	}))
	defer srv.Close()

	cl := Client{cl: http.DefaultClient, webclientAPI: srv.URL + "/api/"}
	_, err = cl.CanvasThreadRoots(t.Context(), "OYP9AAsR28Y", "C06R4HA3ZS8", "UTEST123")
	require.ErrorIs(t, err, errCanvasUserIDMismatch)
	assert.True(t, strings.Contains(err.Error(), "UOTHER"))
}
