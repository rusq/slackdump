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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

func pbGetVarint(b []byte, path ...protowire.Number) (uint64, bool) {
	if len(path) == 0 {
		return 0, false
	}
	cur := b
	for _, want := range path[:len(path)-1] {
		cur = pbGet(cur, want)
		if cur == nil {
			return 0, false
		}
	}
	want := path[len(path)-1]
	rem := cur
	for len(rem) > 0 {
		num, typ, n := protowire.ConsumeTag(rem)
		if n < 0 {
			return 0, false
		}
		rem = rem[n:]
		if num == want && typ == protowire.VarintType {
			val, m := protowire.ConsumeVarint(rem)
			if m < 0 {
				return 0, false
			}
			return val, true
		}
		rem = pbSkip(rem, typ)
		if rem == nil {
			return 0, false
		}
	}
	return 0, false
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

func slackTSFromMicroseconds(us uint64) string {
	return fmt.Sprintf("%d.%06d", us/1_000_000, us%1_000_000)
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
	// The window session ID is generated client-side; controller-init only
	// provides server-side initialisation state.
	return controllerInitResult{UserID: r.UserID}, nil
}

// randSessionID generates a random window session ID (11-char base64url string).
func randSessionID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// editor1Result holds the parsed fields from a load-data/editor/1 protobuf
// response. It exposes the proven thread-related joins from editor/1, but not
// the final Slack root-message timestamp mapping required by messages.list.
type editor1Result struct {
	Section2OYP         string
	CandidateTimestamps []string
	ThreadRecords       []canvasEditor1ThreadRecord
}

type canvasEditor1Entry struct {
	TimestampUS uint64
	Timestamp   string
	BlockIDs    []string
}

type canvasEditor1ThreadRecord struct {
	BlockID          string
	ThreadID         string
	SectionCreatedUS uint64
	SectionCreatedTS string
	SectionEditedUS  uint64
	SectionEditedTS  string
	LatestReplyUS    uint64
	LatestReplyTS    string
	OrphanKey        string
	DocumentID       string
	ParentID         string
	RecordID         string
	RecordKind       string
	IsOrphan         bool
}

var annotationIDRe = regexp.MustCompile(`annotation id="([^"]+)"`)

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

type canvasHistoryMessage struct {
	TS              string                `json:"ts"`
	ThreadTS        string                `json:"thread_ts"`
	SubType         string                `json:"subtype"`
	Text            string                `json:"text"`
	ReplyCount      int                   `json:"reply_count"`
	DocumentComment CanvasDocumentComment `json:"document_comment"`
}

type canvasHistoryResponse struct {
	baseResponse
	Messages []canvasHistoryMessage `json:"messages"`
	HasMore  bool                   `json:"has_more,omitempty"`
}

func decodeEditor1Entry(entry []byte) canvasEditor1Entry {
	var out canvasEditor1Entry
	if us, ok := pbGetVarint(entry, 1); ok {
		out.TimestampUS = us
		out.Timestamp = slackTSFromMicroseconds(us)
	}
	for _, raw := range pbGetAll(entry, 2) {
		if id := pbGetString(raw, 1); id != "" {
			out.BlockIDs = append(out.BlockIDs, id)
		}
	}
	return out
}

func decodeEditor1F7Record(entry []byte) canvasEditor1ThreadRecord {
	var out canvasEditor1ThreadRecord
	out.BlockID = pbGetString(entry, 1)
	out.DocumentID = pbGetString(entry, 6)
	out.ParentID = pbGetString(entry, 7)
	out.OrphanKey = pbGetString(entry, 21)
	out.RecordID = pbGetString(entry, 33)
	out.IsOrphan = out.OrphanKey == "zzzzzz-orphaned-m"
	if us, ok := pbGetVarint(entry, 26); ok {
		out.SectionCreatedUS = us
		out.SectionCreatedTS = slackTSFromMicroseconds(us)
	}
	if us, ok := pbGetVarint(entry, 27); ok {
		out.SectionEditedUS = us
		out.SectionEditedTS = slackTSFromMicroseconds(us)
	}
	if s := pbGetString(entry, 12); s != "" {
		if m := annotationIDRe.FindStringSubmatch(s); len(m) == 2 {
			out.ThreadID = m[1]
			out.RecordKind = "annotated_block"
		}
	}
	if out.ThreadID == "" && out.IsOrphan {
		out.ThreadID = out.BlockID
		out.RecordKind = "orphan_thread"
	}
	return out
}

func mergeEditor1ReplyMetadata(records []canvasEditor1ThreadRecord, replyMeta [][]byte) []canvasEditor1ThreadRecord {
	if len(records) == 0 || len(replyMeta) == 0 {
		return records
	}
	byThreadID := make(map[string]*canvasEditor1ThreadRecord, len(records))
	for i := range records {
		if records[i].ThreadID != "" {
			byThreadID[records[i].ThreadID] = &records[i]
		}
	}
	for _, entry := range replyMeta {
		threadID := pbGetString(entry, 1)
		if threadID == "" {
			continue
		}
		rec := byThreadID[threadID]
		if rec == nil {
			continue
		}
		if v, ok := pbGetVarint(entry, 4); ok {
			rec.LatestReplyUS = v
			rec.LatestReplyTS = slackTSFromMicroseconds(v)
		}
	}
	return records
}

func (cl *Client) fetchEditor1Body(ctx context.Context, oypID, sessionID, userID string) ([]byte, error) {
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
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("editor/1: reading body: %w", err)
	}
	return body, nil
}

// stepEditor1 POSTs to canvas/-/load-data/editor/1 and parses the protobuf response.
// It returns the section-2 OYP ID plus thread-related editor/1 records.
// CandidateTimestamps still reflect legacy f124-derived values and are kept only
// for diagnostics while the real root-message ts mapping remains unresolved.
func (cl *Client) stepEditor1(ctx context.Context, oypID, sessionID, userID string) (editor1Result, error) {
	body, err := cl.fetchEditor1Body(ctx, oypID, sessionID, userID)
	if err != nil {
		return editor1Result{}, err
	}

	section2 := string(pbGet(body, 2, 2, 6, 63))
	if section2 == "" {
		return editor1Result{}, errCanvasMissingSection2OYP
	}

	f124entries := pbGetAll(body, 2, 2, 3, 124)
	slog.Debug("canvas editor/1: f124 entries", "count", len(f124entries))
	for i, entry := range f124entries {
		slog.Debug("canvas editor/1: f124 entry", "i", i, "hex", fmt.Sprintf("%x", entry))
	}
	candidateTimestamps := make([]string, 0, len(f124entries))
	for i, entry := range f124entries {
		// f124 currently appears to track section-created timestamps keyed by
		// commented block IDs rather than the final Slack root-message ts.
		decoded := decodeEditor1Entry(entry)
		slog.Debug("canvas editor/1: decoded f124 entry",
			"i", i,
			"ts_us", decoded.TimestampUS,
			"ts", decoded.Timestamp,
			"block_ids", decoded.BlockIDs)
		if decoded.Timestamp != "" {
			candidateTimestamps = append(candidateTimestamps, decoded.Timestamp)
		}
	}
	if len(candidateTimestamps) == 0 {
		return editor1Result{}, errCanvasMissingThreadTS
	}

	f7entries := pbGetAll(body, 2, 2, 7)
	records := make([]canvasEditor1ThreadRecord, 0, len(f7entries))
	for i, entry := range f7entries {
		rec := decodeEditor1F7Record(entry)
		if rec.BlockID == "" && rec.ThreadID == "" {
			continue
		}
		slog.Debug("canvas editor/1: decoded f7 entry",
			"i", i,
			"block_id", rec.BlockID,
			"thread_id", rec.ThreadID,
			"section_created_ts", rec.SectionCreatedTS,
			"section_edited_ts", rec.SectionEditedTS,
			"record_kind", rec.RecordKind,
			"is_orphan", rec.IsOrphan,
			"record_id", rec.RecordID)
		records = append(records, rec)
	}
	records = mergeEditor1ReplyMetadata(records, pbGetAll(body, 2, 2, 6, 16, 1))

	return editor1Result{
		Section2OYP:         section2,
		CandidateTimestamps: candidateTimestamps,
		ThreadRecords:       records,
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
	slog.Debug("canvas editor/2: reply block IDs", "count", len(blockIDs), "block_ids", blockIDs)
	return blockIDs, nil
}

// messagesListEntry is the JSON structure for the message_ids form field.
type messagesListEntry struct {
	Channel    string   `json:"channel"`
	Timestamps []string `json:"timestamps"`
}

type canvasMessages []CanvasMessage

func (m *canvasMessages) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*m = nil
		return nil
	}
	switch data[0] {
	case '[':
		var msgs []CanvasMessage
		if err := json.Unmarshal(data, &msgs); err != nil {
			return err
		}
		*m = msgs
		return nil
	case '{':
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		var msgs []CanvasMessage
		for _, v := range raw {
			msgs = append(msgs, collectCanvasMessages(v)...)
		}
		*m = msgs
		return nil
	default:
		return fmt.Errorf("unsupported messages JSON shape: %q", data[:1])
	}
}

func collectCanvasMessages(data json.RawMessage) []CanvasMessage {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	switch data[0] {
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil
		}
		var out []CanvasMessage
		for _, item := range raw {
			out = append(out, collectCanvasMessages(item)...)
		}
		return out
	case '{':
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil
		}
		if _, ok := probe["document_comment"]; ok {
			var msg CanvasMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.TS != "" {
				return []CanvasMessage{msg}
			}
		}
		var out []CanvasMessage
		for _, v := range probe {
			out = append(out, collectCanvasMessages(v)...)
		}
		return out
	default:
		return nil
	}
}

type messagesListResponse struct {
	baseResponse
	Messages canvasMessages `json:"messages"`
}

type canvasMessagesListRawResponse struct {
	baseResponse
	Messages     json.RawMessage `json:"messages"`
	MessagesData json.RawMessage `json:"messages_data"`
}

type CanvasMessagesListProbeResult struct {
	Name         string          `json:"name"`
	MessageIDs   string          `json:"message_ids"`
	MessageCount int             `json:"message_count"`
	Keys         []string        `json:"keys,omitempty"`
	Messages     []CanvasMessage `json:"messages,omitempty"`
	Error        string          `json:"error,omitempty"`
}

type CanvasMessagesListDebug struct {
	Section2OYP         string                          `json:"section2_oyp"`
	CandidateTimestamps []string                        `json:"candidate_timestamps"`
	ReplyBlockIDs       []string                        `json:"reply_block_ids"`
	Editor1Matches      []CanvasEditor1Match            `json:"editor1_matches,omitempty"`
	Editor1Subtrees     []CanvasEditor1Subtree          `json:"editor1_subtrees,omitempty"`
	Probes              []CanvasMessagesListProbeResult `json:"probes"`
}

type CanvasEditor1Match struct {
	Path       string `json:"path"`
	ParentPath string `json:"parent_path,omitempty"`
	Value      string `json:"value"`
}

type CanvasEditor1Field struct {
	Field     int    `json:"field"`
	WireType  string `json:"wire_type"`
	String    string `json:"string,omitempty"`
	Varint    uint64 `json:"varint,omitempty"`
	SlackTS   string `json:"slack_ts,omitempty"`
	BytesHex  string `json:"bytes_hex,omitempty"`
	ChildHint string `json:"child_hint,omitempty"`
}

type CanvasEditor1Subtree struct {
	Path   string               `json:"path"`
	Fields []CanvasEditor1Field `json:"fields"`
}

func rawJSONKeys(data []byte) []string {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func extractCanvasMessages(raw canvasMessagesListRawResponse, canvasChannelID string) (canvasMessages, error) {
	if data := bytes.TrimSpace(raw.MessagesData); len(data) > 0 && !bytes.Equal(data, []byte("null")) {
		var md map[string]struct {
			Messages canvasMessages `json:"messages"`
		}
		if err := json.Unmarshal(data, &md); err != nil {
			return nil, err
		}
		if channelData, ok := md[canvasChannelID]; ok {
			return channelData.Messages, nil
		}
	}
	var msgs canvasMessages
	if err := json.Unmarshal(raw.Messages, &msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func pbCollectStringMatches(b []byte, targets map[string]struct{}) []CanvasEditor1Match {
	var out []CanvasEditor1Match
	var walk func([]byte, string, string, []byte)
	walk = func(msg []byte, path string, parentPath string, parentMsg []byte) {
		rem := msg
		for len(rem) > 0 {
			num, typ, n := protowire.ConsumeTag(rem)
			if n < 0 {
				return
			}
			rem = rem[n:]
			fieldPath := fmt.Sprintf("%s.f%d", path, num)
			switch typ {
			case protowire.BytesType:
				val, m := protowire.ConsumeBytes(rem)
				if m < 0 {
					return
				}
				rem = rem[m:]
				if s := string(val); s != "" {
					if _, ok := targets[s]; ok {
						out = append(out, CanvasEditor1Match{
							Path:       fieldPath,
							ParentPath: parentPath,
							Value:      s,
						})
					}
				}
				walk(val, fieldPath, path, msg)
			default:
				rem = pbSkip(rem, typ)
				if rem == nil {
					return
				}
			}
		}
	}
	walk(b, "root", "", nil)
	return out
}

func protobufWireTypeName(typ protowire.Type) string {
	switch typ {
	case protowire.VarintType:
		return "varint"
	case protowire.Fixed64Type:
		return "fixed64"
	case protowire.BytesType:
		return "bytes"
	case protowire.Fixed32Type:
		return "fixed32"
	default:
		return fmt.Sprintf("type_%d", typ)
	}
}

func pbDescribeImmediateFields(msg []byte) []CanvasEditor1Field {
	var out []CanvasEditor1Field
	rem := msg
	for len(rem) > 0 {
		num, typ, n := protowire.ConsumeTag(rem)
		if n < 0 {
			break
		}
		rem = rem[n:]
		f := CanvasEditor1Field{Field: int(num), WireType: protobufWireTypeName(typ)}
		switch typ {
		case protowire.VarintType:
			v, m := protowire.ConsumeVarint(rem)
			if m < 0 {
				return out
			}
			f.Varint = v
			if v > 1_000_000 {
				f.SlackTS = slackTSFromMicroseconds(v)
			}
			rem = rem[m:]
		case protowire.BytesType:
			v, m := protowire.ConsumeBytes(rem)
			if m < 0 {
				return out
			}
			s := string(v)
			if utf8.Valid(v) && s != "" {
				f.String = s
			} else {
				hex := fmt.Sprintf("%x", v)
				if len(hex) > 64 {
					hex = hex[:64]
				}
				f.BytesHex = hex
			}
			if len(v) > 0 && pbGet(v, 1) != nil {
				f.ChildHint = "nested_message"
			}
			rem = rem[m:]
		default:
			next := pbSkip(rem, typ)
			if next == nil {
				return out
			}
			rem = next
		}
		out = append(out, f)
	}
	return out
}

func pbGetParentMessageForMatch(b []byte, target CanvasEditor1Match) []byte {
	var found []byte
	var walk func([]byte, string, string, []byte)
	walk = func(msg []byte, path string, parentPath string, parentMsg []byte) {
		if found != nil {
			return
		}
		rem := msg
		for len(rem) > 0 {
			num, typ, n := protowire.ConsumeTag(rem)
			if n < 0 {
				return
			}
			rem = rem[n:]
			fieldPath := fmt.Sprintf("%s.f%d", path, num)
			switch typ {
			case protowire.BytesType:
				val, m := protowire.ConsumeBytes(rem)
				if m < 0 {
					return
				}
				rem = rem[m:]
				if fieldPath == target.Path && parentPath == target.ParentPath && string(val) == target.Value {
					found = parentMsg
					return
				}
				walk(val, fieldPath, path, msg)
			default:
				rem = pbSkip(rem, typ)
				if rem == nil {
					return
				}
			}
		}
	}
	walk(b, "root", "", nil)
	return found
}

func (cl *Client) stepMessagesListRaw(ctx context.Context, messageIDs string) (canvasMessagesListRawResponse, error) {
	form := url.Values{}
	form.Set("message_ids", messageIDs)
	form.Set("org_wide_aware", "true")
	form.Set("cached_latest_updates", "{}")
	form.Set("_x_reason", "messages-ufm")

	resp, err := cl.PostForm(ctx, "messages.list", form)
	if err != nil {
		return canvasMessagesListRawResponse{}, err
	}
	var r canvasMessagesListRawResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return canvasMessagesListRawResponse{}, fmt.Errorf("messages.list: %w", err)
	}
	if err := r.validate("messages.list"); err != nil {
		return canvasMessagesListRawResponse{}, err
	}
	return r, nil
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
	slog.Debug("canvas messages.list: request", "channel", canvasChannelID, "timestamps", timestamps, "message_ids", string(msgIDs))
	raw, err := cl.stepMessagesListRaw(ctx, string(msgIDs))
	if err != nil {
		return nil, err
	}
	var r messagesListResponse
	r.baseResponse = raw.baseResponse
	msgs, err := extractCanvasMessages(raw, canvasChannelID)
	if err != nil {
		return nil, fmt.Errorf("messages.list: %w", err)
	}
	r.Messages = msgs
	order := make(map[string]int, len(timestamps))
	for i, ts := range timestamps {
		order[ts] = i
	}
	sort.SliceStable(r.Messages, func(i, j int) bool {
		ii, okI := order[r.Messages[i].TS]
		jj, okJ := order[r.Messages[j].TS]
		switch {
		case okI && okJ:
			return ii < jj
		case okI:
			return true
		case okJ:
			return false
		default:
			return r.Messages[i].TS < r.Messages[j].TS
		}
	})
	for i, msg := range r.Messages {
		slog.Debug("canvas messages.list: message",
			"i", i,
			"ts", msg.TS,
			"thread_ts", msg.ThreadTS,
			"thread_id", msg.DocumentComment.ThreadID,
			"reply_count", msg.ReplyCount,
			"text", msg.Text)
	}
	slog.Debug("canvas messages.list: result", "count", len(r.Messages))
	return r.Messages, nil
}

// DebugCanvasMessagesListProbesForCanvas runs the canvas discovery steps and
// tries several candidate messages.list payload shapes to help reverse
// engineer the block-ID to Slack-message mapping.
func (cl *Client) DebugCanvasMessagesListProbesForCanvas(ctx context.Context, oypID, canvasChannelID, userID string) (CanvasMessagesListDebug, error) {
	if _, err := cl.stepControllerInit(ctx); err != nil {
		return CanvasMessagesListDebug{}, err
	}
	sessionID, err := randSessionID()
	if err != nil {
		return CanvasMessagesListDebug{}, fmt.Errorf("canvas: generating session ID: %w", err)
	}
	e1, err := cl.stepEditor1(ctx, oypID, sessionID, userID)
	if err != nil {
		return CanvasMessagesListDebug{}, fmt.Errorf("canvas editor/1: %w", err)
	}
	editor1Body, err := cl.fetchEditor1Body(ctx, oypID, sessionID, userID)
	if err != nil {
		return CanvasMessagesListDebug{}, fmt.Errorf("canvas editor/1 raw: %w", err)
	}
	blockIDs, err := cl.stepEditor2(ctx, sessionID, e1.Section2OYP)
	if err != nil {
		return CanvasMessagesListDebug{}, fmt.Errorf("canvas editor/2: %w", err)
	}

	type probe struct {
		name string
		body any
	}
	var probes []probe
	probes = append(probes, probe{
		name: "timestamps_array",
		body: []messagesListEntry{{
			Channel:    canvasChannelID,
			Timestamps: e1.CandidateTimestamps,
		}},
	})
	probes = append(probes, probe{
		name: "single_object_timestamps",
		body: messagesListEntry{
			Channel:    canvasChannelID,
			Timestamps: e1.CandidateTimestamps,
		},
	})
	probes = append(probes,
		probe{name: "thread_ids_array", body: []map[string]any{{"channel": canvasChannelID, "thread_ids": blockIDs}}},
		probe{name: "message_ids_array", body: []map[string]any{{"channel": canvasChannelID, "message_ids": blockIDs}}},
		probe{name: "ids_array", body: []map[string]any{{"channel": canvasChannelID, "ids": blockIDs}}},
		probe{name: "messages_thread_id_array", body: []map[string]any{{"channel": canvasChannelID, "messages": mapsOf("thread_id", blockIDs)}}},
		probe{name: "messages_block_id_array", body: []map[string]any{{"channel": canvasChannelID, "messages": mapsOf("block_id", blockIDs)}}},
	)

	out := CanvasMessagesListDebug{
		Section2OYP:         e1.Section2OYP,
		CandidateTimestamps: e1.CandidateTimestamps,
		ReplyBlockIDs:       blockIDs,
		Editor1Matches:      pbCollectStringMatches(editor1Body, toSet(blockIDs)),
		Probes:              make([]CanvasMessagesListProbeResult, 0, len(probes)),
	}
	seenSubtrees := make(map[string]struct{})
	for _, match := range out.Editor1Matches {
		parent := match.ParentPath
		if parent == "" {
			continue
		}
		key := parent + "\x00" + match.Value
		if _, ok := seenSubtrees[key]; ok {
			continue
		}
		seenSubtrees[key] = struct{}{}
		parentMsg := pbGetParentMessageForMatch(editor1Body, match)
		if parentMsg == nil {
			continue
		}
		out.Editor1Subtrees = append(out.Editor1Subtrees, CanvasEditor1Subtree{
			Path:   parent,
			Fields: pbDescribeImmediateFields(parentMsg),
		})
	}
	for _, p := range probes {
		msgIDs, err := json.Marshal(p.body)
		if err != nil {
			out.Probes = append(out.Probes, CanvasMessagesListProbeResult{Name: p.name, Error: err.Error()})
			continue
		}
		res := CanvasMessagesListProbeResult{Name: p.name, MessageIDs: string(msgIDs)}
		raw, err := cl.stepMessagesListRaw(ctx, string(msgIDs))
		if err != nil {
			res.Error = err.Error()
			out.Probes = append(out.Probes, res)
			continue
		}
		if len(bytes.TrimSpace(raw.MessagesData)) > 0 {
			res.Keys = rawJSONKeys(raw.MessagesData)
		} else {
			res.Keys = rawJSONKeys(raw.Messages)
		}
		msgs, err := extractCanvasMessages(raw, canvasChannelID)
		if err != nil {
			res.Error = err.Error()
			out.Probes = append(out.Probes, res)
			continue
		}
		res.Messages = msgs
		res.MessageCount = len(res.Messages)
		out.Probes = append(out.Probes, res)
	}
	return out, nil
}

func mapsOf(key string, values []string) []map[string]string {
	out := make([]map[string]string, 0, len(values))
	for _, v := range values {
		out = append(out, map[string]string{key: v})
	}
	return out
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}

func parentPath(path string) string {
	i := strings.LastIndex(path, ".")
	if i == -1 {
		return "root"
	}
	return path[:i]
}

// canvasChannelFromFileID derives the dedicated canvas channel ID from a file ID.
// Valid canvas channels reuse the file suffix with a leading C instead of F.
func canvasChannelFromFileID(fileID string) string {
	if len(fileID) < 2 || fileID[0] != 'F' {
		return ""
	}
	return "C" + fileID[1:]
}

func (cl *Client) conversationsHistoryForCanvas(ctx context.Context, channelID string) ([]canvasHistoryMessage, error) {
	const ep = "conversations.history"
	type form struct {
		BaseRequest
		Channel string `json:"channel"`
		Limit   int    `json:"limit"`
		Cursor  string `json:"cursor,omitempty"`
		WebClientFields
	}

	req := form{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channelID,
		Limit:           1000,
		WebClientFields: webclientReason("messages-ufm"),
	}

	var out []canvasHistoryMessage
	for {
		resp, err := cl.PostFormRaw(ctx, cl.webapiURL(ep), values(req, true))
		if err != nil {
			return nil, err
		}
		var r canvasHistoryResponse
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, fmt.Errorf("%s: %w", ep, err)
		}
		if err := r.validate(ep); err != nil {
			return nil, err
		}
		out = append(out, r.Messages...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		req.Cursor = r.ResponseMetadata.NextCursor
	}
	return out, nil
}

// CanvasThreadRoots returns the root messages for all comment threads on a
// canvas file. fileID is the Slack file ID, for example F06R4HA3ZS8.
func (cl *Client) CanvasThreadRoots(ctx context.Context, fileID string) ([]CanvasMessage, error) {
	canvasChannelID := canvasChannelFromFileID(fileID)
	if canvasChannelID == "" {
		return nil, fmt.Errorf("canvas: invalid file ID %q", fileID)
	}
	msgs, err := cl.conversationsHistoryForCanvas(ctx, canvasChannelID)
	if err != nil {
		return nil, fmt.Errorf("canvas conversations.history: %w", err)
	}
	roots := make([]CanvasMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg.SubType != "document_comment_root" {
			continue
		}
		threadTS := msg.ThreadTS
		if threadTS == "" {
			threadTS = msg.TS
		}
		roots = append(roots, CanvasMessage{
			TS:              msg.TS,
			ThreadTS:        threadTS,
			Text:            msg.Text,
			ReplyCount:      msg.ReplyCount,
			DocumentComment: msg.DocumentComment,
		})
	}
	return roots, nil
}
