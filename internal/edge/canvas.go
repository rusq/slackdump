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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// quip.* API

type quipLookupForm struct {
	BaseRequest
	FileIDs string `json:"file_ids"`
	WebClientFields
}

type quipLookupResponse struct {
	baseResponse
	Lookup map[string]string `json:"lookup"`
}

var (
	errCanvasMissingSession     = errors.New("canvas controller-init: missing session ID")
	errCanvasMissingSection2OYP = errors.New("canvas editor/1: missing section-2 OYP ID")
	errCanvasMissingThreadTS    = errors.New("canvas editor/1: no thread timestamps found")
	errCanvasMalformedEditor2   = errors.New("canvas editor/2: malformed response")
	errCanvasUserIDMismatch     = errors.New("canvas controller-init: user ID mismatch")
)

// QuipLookupThreadIDs maps Slack file IDs to Quip/OYP document IDs used
// in canvas load-data requests. Returns a map of fileID → OYP ID.
func (cl *Client) QuipLookupThreadIDs(ctx context.Context, fileID ...string) (map[string]string, error) {
	if len(fileID) == 0 {
		return map[string]string{}, nil
	}
	const ep = "quip.lookupThreadIds"
	form := quipLookupForm{
		BaseRequest:     BaseRequest{Token: cl.token},
		FileIDs:         strings.Join(fileID, ","),
		WebClientFields: webclientReason("fetch-quip-ids"),
	}
	resp, err := cl.Post(ctx, ep, form)
	if err != nil {
		return nil, err
	}
	var r quipLookupResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, fmt.Errorf("%s: %w", ep, err)
	}
	if err := r.validate(ep); err != nil {
		return nil, err
	}
	return r.Lookup, nil
}

// encodeEditor1Request encodes the request_binary protobuf for load-data/editor/1.
func encodeEditor1Request(oypID string) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, oypID)
	sub := protowire.AppendTag(nil, 1, protowire.VarintType)
	sub = protowire.AppendVarint(sub, 1)
	sub = protowire.AppendTag(sub, 2, protowire.BytesType)
	sub = protowire.AppendString(sub, oypID)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, sub)
	b = protowire.AppendTag(b, 5, protowire.BytesType)
	b = protowire.AppendString(b, "editor")
	b = protowire.AppendTag(b, 6, protowire.VarintType)
	b = protowire.AppendVarint(b, 1)
	return b
}

// encodeEditor2Request encodes the request_binary protobuf for load-data/editor/2.
func encodeEditor2Request(sessionID, oypID string) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, sessionID)
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, oypID)
	b = protowire.AppendTag(b, 5, protowire.BytesType)
	b = protowire.AppendString(b, "editor")
	b = protowire.AppendTag(b, 6, protowire.VarintType)
	b = protowire.AppendVarint(b, 2)
	return b
}

// pbSkip advances b past the value for the given wire type.
// Returns nil if the type is unknown or data is malformed.
func pbSkip(b []byte, typ protowire.Type) []byte {
	switch typ {
	case protowire.VarintType:
		_, m := protowire.ConsumeVarint(b)
		if m < 0 {
			return nil
		}
		return b[m:]
	case protowire.Fixed64Type:
		_, m := protowire.ConsumeFixed64(b)
		if m < 0 {
			return nil
		}
		return b[m:]
	case protowire.BytesType:
		_, m := protowire.ConsumeBytes(b)
		if m < 0 {
			return nil
		}
		return b[m:]
	case protowire.Fixed32Type:
		_, m := protowire.ConsumeFixed32(b)
		if m < 0 {
			return nil
		}
		return b[m:]
	default:
		return nil
	}
}

// pbGet traverses a protobuf-encoded byte slice following the given field-number
// path. Returns the raw bytes of the leaf field, or nil if not found.
// Each step in path must be a BytesType (length-delimited) field.
func pbGet(b []byte, path ...protowire.Number) []byte {
	cur := b
	for _, want := range path {
		var found []byte
		rem := cur
		for len(rem) > 0 {
			num, typ, n := protowire.ConsumeTag(rem)
			if n < 0 {
				break
			}
			rem = rem[n:]
			if num == want && typ == protowire.BytesType {
				val, m := protowire.ConsumeBytes(rem)
				if m < 0 {
					break
				}
				found = val
				break
			}
			rem = pbSkip(rem, typ)
			if rem == nil {
				break
			}
		}
		if found == nil {
			return nil
		}
		cur = found
	}
	return cur
}

// pbGetAll returns all occurrences of a repeated field at the given path.
// All but the last element of path are traversed via pbGet (first match only).
// The last element is collected exhaustively.
func pbGetAll(b []byte, path ...protowire.Number) [][]byte {
	if len(path) == 0 {
		return nil
	}
	cur := b
	for _, want := range path[:len(path)-1] {
		next := pbGet(cur, want)
		if next == nil {
			return nil
		}
		cur = next
	}
	want := path[len(path)-1]
	var result [][]byte
	rem := cur
	for len(rem) > 0 {
		num, typ, n := protowire.ConsumeTag(rem)
		if n < 0 {
			break
		}
		rem = rem[n:]
		if num == want && typ == protowire.BytesType {
			val, m := protowire.ConsumeBytes(rem)
			if m < 0 {
				break
			}
			result = append(result, val)
			rem = rem[m:]
			continue
		}
		rem = pbSkip(rem, typ)
		if rem == nil {
			break
		}
	}
	return result
}

// pbFirstString returns the string value of the first BytesType field in b,
// or "" if none is found.
func pbFirstString(b []byte) string {
	for len(b) > 0 {
		_, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return ""
		}
		b = b[n:]
		if typ == protowire.BytesType {
			val, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return ""
			}
			return string(val)
		}
		b = pbSkip(b, typ)
		if b == nil {
			return ""
		}
	}
	return ""
}

func pbGetString(b []byte, path ...protowire.Number) string {
	return string(pbGet(b, path...))
}

// pbFindString walks a protobuf message and returns the first bytes field that
// satisfies match. Bytes fields are recursively searched because some canvas
// responses wrap values in nested sub-messages.
func pbFindString(b []byte, match func(string) bool) string {
	for len(b) > 0 {
		_, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return ""
		}
		b = b[n:]
		if typ != protowire.BytesType {
			b = pbSkip(b, typ)
			if b == nil {
				return ""
			}
			continue
		}
		val, m := protowire.ConsumeBytes(b)
		if m < 0 {
			return ""
		}
		b = b[m:]
		if s := string(val); match(s) {
			return s
		}
		if nested := pbFindString(val, match); nested != "" {
			return nested
		}
	}
	return ""
}

func looksLikeSlackTS(s string) bool {
	if len(s) < 3 {
		return false
	}
	dot := strings.IndexByte(s, '.')
	if dot <= 0 || dot == len(s)-1 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// canvasBaseURL returns the workspace base URL derived from webclientAPI.
// e.g. "https://myteam.slack.com/api/" → "https://myteam.slack.com/".
func (cl *Client) canvasBaseURL() string {
	return strings.TrimSuffix(cl.webclientAPI, "api/")
}

type canvasControllerInitResponse struct {
	InitOptions string `json:"init_options"`
	UserID      string `json:"user_id"`
}

type controllerInitResult struct {
	SessionID string
	UserID    string
}

func encodeControllerInitOptions(sessionID, docID string) []byte {
	var session []byte
	session = protowire.AppendTag(session, 1, protowire.BytesType)
	session = protowire.AppendString(session, sessionID)
	if docID != "" {
		session = protowire.AppendTag(session, 2, protowire.BytesType)
		session = protowire.AppendString(session, docID)
	}
	session = protowire.AppendTag(session, 3, protowire.BytesType)
	session = protowire.AppendString(session, "")

	var b []byte
	b = protowire.AppendTag(b, 43, protowire.BytesType)
	b = protowire.AppendBytes(b, session)
	b = protowire.AppendTag(b, 52, protowire.BytesType)
	b = protowire.AppendString(b, sessionID)
	return b
}

func decodeControllerInitOptions(initOptions string) (string, error) {
	body, err := base64.StdEncoding.DecodeString(initOptions)
	if err != nil {
		return "", fmt.Errorf("canvas controller-init: decoding init_options: %w", err)
	}
	sessionID := pbGetString(body, 43, 1)
	if sessionID == "" {
		return "", errCanvasMissingSession
	}
	if duplicate := pbGetString(body, 52); duplicate != "" && duplicate != sessionID {
		return "", fmt.Errorf("canvas controller-init: session mismatch: %q != %q", sessionID, duplicate)
	}
	return sessionID, nil
}

func (cl *Client) stepControllerInit(ctx context.Context) (controllerInitResult, error) {
	endpoint := cl.canvasBaseURL() + "canvas/collab/controller-init?format=map"

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("token", cl.token); err != nil {
		return controllerInitResult{}, fmt.Errorf("canvas controller-init: writing form: %w", err)
	}
	if err := w.Close(); err != nil {
		return controllerInitResult{}, fmt.Errorf("canvas controller-init: closing form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, cl.recorder(bytes.NewReader(body.Bytes())))
	if err != nil {
		return controllerInitResult{}, err
	}
	defer cl.record([]byte("\n\n"))
	req.Header.Set(hdrContentType, w.FormDataContentType())

	resp, err := do(ctx, cl.cl, req)
	if err != nil {
		return controllerInitResult{}, err
	}
	defer resp.Body.Close()

	var r canvasControllerInitResponse
	if err := json.NewDecoder(cl.recorder(resp.Body)).Decode(&r); err != nil {
		return controllerInitResult{}, fmt.Errorf("canvas controller-init: %w", err)
	}
	sessionID, err := decodeControllerInitOptions(r.InitOptions)
	if err != nil {
		return controllerInitResult{}, err
	}
	return controllerInitResult{
		SessionID: sessionID,
		UserID:    r.UserID,
	}, nil
}

// editor1Result holds the parsed fields from a load-data/editor/1 protobuf response.
type editor1Result struct {
	Section2OYP      string
	ThreadTimestamps []string
}

// CanvasDocumentComment holds the document_comment subfields of a canvas message.
type CanvasDocumentComment struct {
	ThreadID string   `json:"thread_id"`
	Authors  []string `json:"authors"`
}

// CanvasMessage is a thread root message returned by messages.list for a canvas.
type CanvasMessage struct {
	TS              string                `json:"ts"`
	ThreadTS        string                `json:"thread_ts"`
	Text            string                `json:"text"`
	ReplyCount      int                   `json:"reply_count"`
	DocumentComment CanvasDocumentComment `json:"document_comment"`
}

// stepEditor1 POSTs to canvas/-/load-data/editor/1 and parses the protobuf response.
// It returns the section-2 OYP ID and the thread-root timestamps embedded in f3.f124[].
func (cl *Client) stepEditor1(ctx context.Context, oypID, sessionID, userID string) (editor1Result, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	endpoint := cl.canvasBaseURL() + "canvas/-/load-data/editor/1?_x_version_ts=" + ts

	form := url.Values{}
	form.Set("_resource_bundle", "collab_controller")
	form.Set("_user_id", userID)
	form.Set("_version", "10")
	form.Set("_window_session_id", sessionID)
	form.Set("request_binary", base64.StdEncoding.EncodeToString(encodeEditor1Request(oypID)))

	resp, err := cl.PostFormRaw(ctx, endpoint, form)
	if err != nil {
		return editor1Result{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return editor1Result{}, fmt.Errorf("editor/1: reading body: %w", err)
	}

	section2 := string(pbGet(body, 6, 63))
	if section2 == "" {
		return editor1Result{}, errCanvasMissingSection2OYP
	}

	f124entries := pbGetAll(body, 3, 124)
	if len(f124entries) > 0 {
		// Log first entry to help identify the timestamp subfield during live runs.
		log.Printf("canvas editor/1: first f124 entry (hex): %x", f124entries[0])
	}
	timestamps := make([]string, 0, len(f124entries))
	for _, entry := range f124entries {
		// The exact timestamp subfield inside f124 is still being reverse
		// engineered; match the first Slack timestamp-shaped string rather than
		// the first arbitrary bytes field.
		if s := pbFindString(entry, looksLikeSlackTS); s != "" {
			timestamps = append(timestamps, s)
		}
	}
	if len(timestamps) == 0 {
		return editor1Result{}, errCanvasMissingThreadTS
	}

	return editor1Result{
		Section2OYP:      section2,
		ThreadTimestamps: timestamps,
	}, nil
}

// stepEditor2 POSTs to canvas/-/load-data/editor/2 and returns the block IDs
// that have at least one reply (f2.f2.f55.f8[].f3).
func (cl *Client) stepEditor2(ctx context.Context, sessionID, section2OYP string) ([]string, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	endpoint := cl.canvasBaseURL() + "canvas/-/load-data/editor/2?_x_version_ts=" + ts

	form := url.Values{}
	form.Set("_resource_bundle", "collab_controller")
	form.Set("_version", "10")
	form.Set("_window_session_id", sessionID)
	form.Set("request_binary", base64.StdEncoding.EncodeToString(encodeEditor2Request(sessionID, section2OYP)))

	resp, err := cl.PostFormRaw(ctx, endpoint, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("editor/2: reading body: %w", err)
	}
	if pbGet(body, 2, 2, 55) == nil {
		return nil, errCanvasMalformedEditor2
	}

	f8entries := pbGetAll(body, 2, 2, 55, 8)
	blockIDs := make([]string, 0, len(f8entries))
	for _, entry := range f8entries {
		if id := pbGet(entry, 3); id != nil {
			blockIDs = append(blockIDs, string(id))
		}
	}
	return blockIDs, nil
}

// messagesListEntry is the JSON structure for the message_ids form field.
type messagesListEntry struct {
	Channel    string   `json:"channel"`
	Timestamps []string `json:"timestamps"`
}

type messagesListResponse struct {
	baseResponse
	Messages []CanvasMessage `json:"messages"`
}

// stepMessagesList POSTs to messages.list and returns the thread root messages
// for the given canvas channel and timestamps.
func (cl *Client) stepMessagesList(ctx context.Context, canvasChannelID string, timestamps []string) ([]CanvasMessage, error) {
	msgIDs, err := json.Marshal([]messagesListEntry{{
		Channel:    canvasChannelID,
		Timestamps: timestamps,
	}})
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("message_ids", string(msgIDs))
	form.Set("org_wide_aware", "true")
	form.Set("cached_latest_updates", "{}")
	form.Set("_x_reason", "messages-ufm")

	resp, err := cl.PostForm(ctx, "messages.list", form)
	if err != nil {
		return nil, err
	}
	var r messagesListResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, fmt.Errorf("messages.list: %w", err)
	}
	if err := r.validate("messages.list"); err != nil {
		return nil, err
	}
	return r.Messages, nil
}

// CanvasThreadRoots returns the thread root messages for a canvas document.
// oypID is the section-1 OYP ID from QuipLookupThreadIDs.
// canvasChannelID is the canvas's dedicated channel (e.g. C06R4HA3ZS8).
// userID is the authenticated user's ID.
func (cl *Client) CanvasThreadRoots(ctx context.Context, oypID, canvasChannelID, userID string) ([]CanvasMessage, error) {
	init, err := cl.stepControllerInit(ctx)
	if err != nil {
		return nil, err
	}
	if init.UserID != "" && userID != "" && init.UserID != userID {
		return nil, fmt.Errorf("%w: %q != %q", errCanvasUserIDMismatch, init.UserID, userID)
	}

	e1, err := cl.stepEditor1(ctx, oypID, init.SessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("canvas editor/1: %w", err)
	}

	if _, err := cl.stepEditor2(ctx, init.SessionID, e1.Section2OYP); err != nil {
		return nil, fmt.Errorf("canvas editor/2: %w", err)
	}

	msgs, err := cl.stepMessagesList(ctx, canvasChannelID, e1.ThreadTimestamps)
	if err != nil {
		return nil, fmt.Errorf("canvas messages.list: %w", err)
	}
	return msgs, nil
}
