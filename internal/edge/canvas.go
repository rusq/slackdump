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
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/rusq/slack"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/rusq/slackdump/v4/internal/structures"
)

var (
	errCanvasMissingOYPID       = errors.New("canvas quip.lookupThreadIds: missing document ID")
	errCanvasMissingUserID      = errors.New("canvas controller-init: missing user ID")
	errCanvasMissingThreadTable = errors.New("canvas editor/1: missing thread metadata table")
)

// CanvasDocumentComment holds the document_comment subfields of a canvas message.
type CanvasDocumentComment struct {
	ThreadID string   `json:"thread_id"`
	Authors  []string `json:"authors"`
}

// CanvasMessage is a root message for a canvas comment thread.
type CanvasMessage struct {
	TS              string                `json:"ts"`
	ThreadTS        string                `json:"thread_ts"`
	Text            string                `json:"text"`
	ReplyCount      int                   `json:"reply_count"`
	DocumentComment CanvasDocumentComment `json:"document_comment"`
	// Message retains the complete Slack root for archival callers. It is
	// intentionally excluded from JSON so tools edge keeps its existing
	// diagnostic output schema.
	Message slack.Message `json:"-"`
}

type canvasThreadMetadata struct {
	ThreadID        string
	ReplyCount      int
	LatestMessageTS string
}

type canvasAPIMessage struct {
	TS              string                `json:"ts"`
	ThreadTS        string                `json:"thread_ts"`
	SubType         string                `json:"subtype"`
	Text            string                `json:"text"`
	ReplyCount      int                   `json:"reply_count"`
	DocumentComment CanvasDocumentComment `json:"document_comment"`
	Message         slack.Message         `json:"-"`
}

func (m *canvasAPIMessage) UnmarshalJSON(data []byte) error {
	type wireMessage canvasAPIMessage
	var wire wireMessage
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var message slack.Message
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}
	*m = canvasAPIMessage(wire)
	m.Message = message
	return nil
}

type canvasRepliesResponse struct {
	baseResponse
	Messages []canvasAPIMessage `json:"messages"`
	HasMore  bool               `json:"has_more,omitempty"`
}

type quipLookupForm struct {
	BaseRequest
	FileIDs string `json:"file_ids"`
	WebClientFields
}

type quipLookupResponse struct {
	baseResponse
	Lookup map[string]string `json:"lookup"`
}

// quipLookupThreadIDs maps Slack canvas file IDs to the Quip/OYP document IDs
// used by the canvas load-data endpoints.
func (cl *Client) quipLookupThreadIDs(ctx context.Context, fileIDs ...string) (map[string]string, error) {
	if len(fileIDs) == 0 {
		return map[string]string{}, nil
	}
	const ep = "quip.lookupThreadIds"
	resp, err := cl.Post(ctx, ep, quipLookupForm{
		BaseRequest:     BaseRequest{Token: cl.token},
		FileIDs:         strings.Join(fileIDs, ","),
		WebClientFields: webclientReason("fetch-quip-ids"),
	})
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

// canvasBaseURL returns the workspace base URL derived from webclientAPI.
// For example, "https://myteam.slack.com/api/" becomes
// "https://myteam.slack.com/".
func (cl *Client) canvasBaseURL() string {
	return strings.TrimSuffix(cl.webclientAPI, "api/")
}

type canvasControllerInitResponse struct {
	UserID string `json:"user_id"`
}

func (cl *Client) canvasControllerInit(ctx context.Context) (string, error) {
	endpoint := cl.canvasBaseURL() + "canvas/collab/controller-init?format=map"

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("token", cl.token); err != nil {
		return "", fmt.Errorf("canvas controller-init: writing form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("canvas controller-init: closing form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, cl.recorder(bytes.NewReader(body.Bytes())))
	if err != nil {
		return "", err
	}
	defer cl.record([]byte("\n\n"))
	req.Header.Set(hdrContentType, writer.FormDataContentType())

	resp, err := do(ctx, cl.cl, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var r canvasControllerInitResponse
	if err := json.NewDecoder(cl.recorder(resp.Body)).Decode(&r); err != nil {
		return "", fmt.Errorf("canvas controller-init: decoding response: %w", err)
	}
	if r.UserID == "" {
		return "", errCanvasMissingUserID
	}
	return r.UserID, nil
}

func randomCanvasSessionID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func encodeEditor1Request(oypID string) []byte {
	var request []byte
	request = protowire.AppendTag(request, 1, protowire.BytesType)
	request = protowire.AppendString(request, oypID)

	var document []byte
	document = protowire.AppendTag(document, 1, protowire.VarintType)
	document = protowire.AppendVarint(document, 1)
	document = protowire.AppendTag(document, 2, protowire.BytesType)
	document = protowire.AppendString(document, oypID)

	request = protowire.AppendTag(request, 3, protowire.BytesType)
	request = protowire.AppendBytes(request, document)
	request = protowire.AppendTag(request, 5, protowire.BytesType)
	request = protowire.AppendString(request, "editor")
	request = protowire.AppendTag(request, 6, protowire.VarintType)
	request = protowire.AppendVarint(request, 1)
	return request
}

func pbSkip(b []byte, typ protowire.Type) []byte {
	var n int
	switch typ {
	case protowire.VarintType:
		_, n = protowire.ConsumeVarint(b)
	case protowire.Fixed64Type:
		_, n = protowire.ConsumeFixed64(b)
	case protowire.BytesType:
		_, n = protowire.ConsumeBytes(b)
	case protowire.Fixed32Type:
		_, n = protowire.ConsumeFixed32(b)
	default:
		return nil
	}
	if n < 0 {
		return nil
	}
	return b[n:]
}

// pbGet follows a path of length-delimited protobuf fields and returns the
// first matching value.
func pbGet(b []byte, path ...protowire.Number) []byte {
	current := b
	for _, wanted := range path {
		var found []byte
		for remaining := current; len(remaining) > 0; {
			number, typ, n := protowire.ConsumeTag(remaining)
			if n < 0 {
				return nil
			}
			remaining = remaining[n:]
			if number == wanted && typ == protowire.BytesType {
				value, m := protowire.ConsumeBytes(remaining)
				if m < 0 {
					return nil
				}
				found = value
				break
			}
			remaining = pbSkip(remaining, typ)
			if remaining == nil {
				return nil
			}
		}
		if found == nil {
			return nil
		}
		current = found
	}
	return current
}

// pbGetAll returns every occurrence of the final length-delimited field in a
// protobuf path.
func pbGetAll(b []byte, path ...protowire.Number) [][]byte {
	if len(path) == 0 {
		return nil
	}
	current := b
	for _, wanted := range path[:len(path)-1] {
		current = pbGet(current, wanted)
		if current == nil {
			return nil
		}
	}

	wanted := path[len(path)-1]
	var values [][]byte
	for remaining := current; len(remaining) > 0; {
		number, typ, n := protowire.ConsumeTag(remaining)
		if n < 0 {
			return nil
		}
		remaining = remaining[n:]
		if number == wanted && typ == protowire.BytesType {
			value, m := protowire.ConsumeBytes(remaining)
			if m < 0 {
				return nil
			}
			values = append(values, value)
			remaining = remaining[m:]
			continue
		}
		remaining = pbSkip(remaining, typ)
		if remaining == nil {
			return nil
		}
	}
	return values
}

func pbGetString(b []byte, field protowire.Number) string {
	return string(pbGet(b, field))
}

func pbGetVarint(b []byte, wanted protowire.Number) (uint64, bool) {
	for remaining := b; len(remaining) > 0; {
		number, typ, n := protowire.ConsumeTag(remaining)
		if n < 0 {
			return 0, false
		}
		remaining = remaining[n:]
		if number == wanted && typ == protowire.VarintType {
			value, m := protowire.ConsumeVarint(remaining)
			if m < 0 {
				return 0, false
			}
			return value, true
		}
		remaining = pbSkip(remaining, typ)
		if remaining == nil {
			return 0, false
		}
	}
	return 0, false
}

func slackTSFromMicroseconds(us uint64) string {
	return fmt.Sprintf("%d.%06d", us/1_000_000, us%1_000_000)
}

func decodeCanvasThreadMetadata(body []byte) ([]canvasThreadMetadata, error) {
	table := pbGet(body, 2, 2, 6, 16)
	if table == nil {
		return nil, errCanvasMissingThreadTable
	}

	rawEntries := pbGetAll(table, 1)
	entries := make([]canvasThreadMetadata, 0, len(rawEntries))
	for i, raw := range rawEntries {
		threadID := pbGetString(raw, 1)
		if threadID == "" {
			return nil, fmt.Errorf("canvas editor/1: thread metadata entry %d: missing thread ID", i)
		}
		replyCount, _ := pbGetVarint(raw, 2)
		if replyCount > uint64(^uint(0)>>1) {
			return nil, fmt.Errorf("canvas editor/1: thread metadata entry %d: reply count overflow", i)
		}
		latestUS, ok := pbGetVarint(raw, 4)
		if !ok || latestUS == 0 {
			return nil, fmt.Errorf("canvas editor/1: thread metadata entry %d: missing latest message timestamp", i)
		}
		entries = append(entries, canvasThreadMetadata{
			ThreadID:        threadID,
			ReplyCount:      int(replyCount),
			LatestMessageTS: slackTSFromMicroseconds(latestUS),
		})
	}
	return entries, nil
}

func (cl *Client) canvasEditor1(ctx context.Context, oypID, sessionID, userID string) ([]canvasThreadMetadata, error) {
	endpoint := cl.canvasBaseURL() + "canvas/-/load-data/editor/1?_x_version_ts=" +
		strconv.FormatInt(time.Now().UnixMilli(), 10)

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
		return nil, fmt.Errorf("canvas editor/1: reading response: %w", err)
	}
	return decodeCanvasThreadMetadata(body)
}

func (cl *Client) canvasConversationReplies(ctx context.Context, channelID, timestamp string) ([]canvasAPIMessage, error) {
	const ep = "conversations.replies"
	type form struct {
		BaseRequest
		Channel string `json:"channel"`
		TS      string `json:"ts"`
		Limit   int    `json:"limit"`
		Cursor  string `json:"cursor,omitempty"`
		WebClientFields
	}

	request := form{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channelID,
		TS:              timestamp,
		Limit:           1000,
		WebClientFields: webclientReason("canvas-comments"),
	}

	var messages []canvasAPIMessage
	for {
		resp, err := cl.PostFormRaw(ctx, cl.webapiURL(ep), values(request, true))
		if err != nil {
			return nil, err
		}
		var r canvasRepliesResponse
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, fmt.Errorf("%s: %w", ep, err)
		}
		if err := r.validate(ep); err != nil {
			return nil, err
		}
		messages = append(messages, r.Messages...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		request.Cursor = r.ResponseMetadata.NextCursor
	}
	return messages, nil
}

func resolvedCanvasRootTS(metadata canvasThreadMetadata, probe []canvasAPIMessage) (string, error) {
	if metadata.ReplyCount == 0 {
		return metadata.LatestMessageTS, nil
	}
	for _, message := range probe {
		if message.TS == metadata.LatestMessageTS && message.ThreadTS != "" {
			return message.ThreadTS, nil
		}
	}
	return "", fmt.Errorf(
		"canvas thread %q: latest reply %s did not resolve to a root",
		metadata.ThreadID,
		metadata.LatestMessageTS,
	)
}

func canvasRootMessage(metadata canvasThreadMetadata, rootTS string, messages []canvasAPIMessage) (CanvasMessage, error) {
	for _, message := range messages {
		if message.TS != rootTS || message.SubType != structures.SubTypeDocumentCommentRoot {
			continue
		}
		if message.DocumentComment.ThreadID != "" && message.DocumentComment.ThreadID != metadata.ThreadID {
			return CanvasMessage{}, fmt.Errorf(
				"canvas root %s: thread ID mismatch: got %q, want %q",
				rootTS,
				message.DocumentComment.ThreadID,
				metadata.ThreadID,
			)
		}
		if message.DocumentComment.ThreadID == "" {
			message.DocumentComment.ThreadID = metadata.ThreadID
		}
		threadTS := message.ThreadTS
		if threadTS == "" {
			threadTS = message.TS
		}
		message.Message.ThreadTimestamp = threadTS
		return CanvasMessage{
			TS:              message.TS,
			ThreadTS:        threadTS,
			Text:            message.Text,
			ReplyCount:      message.ReplyCount,
			DocumentComment: message.DocumentComment,
			Message:         message.Message,
		}, nil
	}
	return CanvasMessage{}, fmt.Errorf("canvas thread %q: root %s not found", metadata.ThreadID, rootTS)
}

// CanvasThreadRoots returns the root messages for the current comment threads
// on a canvas file. fileID is the Slack file ID, for example F06R4HA3ZS8.
func (cl *Client) CanvasThreadRoots(ctx context.Context, fileID string) ([]CanvasMessage, error) {
	ctx, task := trace.NewTask(ctx, "CanvasThreadRoots")
	defer task.End()

	channelID, ok := structures.CanvasChannelID(fileID)
	if channelID == "" || !ok {
		return nil, fmt.Errorf("canvas: invalid file ID %q", fileID)
	}

	lookup, err := cl.quipLookupThreadIDs(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("canvas quip.lookupThreadIds: %w", err)
	}
	oypID := lookup[fileID]
	if oypID == "" {
		return nil, fmt.Errorf("%w for file %q", errCanvasMissingOYPID, fileID)
	}

	userID, err := cl.canvasControllerInit(ctx)
	if err != nil {
		return nil, err
	}
	sessionID, err := randomCanvasSessionID()
	if err != nil {
		return nil, fmt.Errorf("canvas: generating window session ID: %w", err)
	}
	metadata, err := cl.canvasEditor1(ctx, oypID, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("canvas editor/1: %w", err)
	}

	slog.DebugContext(ctx, "canvas editor/1: thread metadata", "count", len(metadata))
	roots := make([]CanvasMessage, 0, len(metadata))
	seen := make(map[string]struct{}, len(metadata))
	for _, entry := range metadata {
		rootTS := entry.LatestMessageTS
		if entry.ReplyCount > 0 {
			probe, err := cl.canvasConversationReplies(ctx, channelID, entry.LatestMessageTS)
			if err != nil {
				return nil, fmt.Errorf("canvas thread %q: resolving root: %w", entry.ThreadID, err)
			}
			rootTS, err = resolvedCanvasRootTS(entry, probe)
			if err != nil {
				return nil, err
			}
		}
		if _, ok := seen[rootTS]; ok {
			continue
		}
		messages, err := cl.canvasConversationReplies(ctx, channelID, rootTS)
		if err != nil {
			return nil, fmt.Errorf("canvas thread %q: fetching root: %w", entry.ThreadID, err)
		}
		root, err := canvasRootMessage(entry, rootTS, messages)
		if err != nil {
			return nil, err
		}
		seen[rootTS] = struct{}{}
		roots = append(roots, root)
		slog.DebugContext(ctx, "canvas thread root",
			"thread_id", entry.ThreadID,
			"latest_message_ts", entry.LatestMessageTS,
			"root_ts", rootTS,
			"reply_count", root.ReplyCount)
	}
	return roots, nil
}
