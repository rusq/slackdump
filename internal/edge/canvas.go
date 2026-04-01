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
	"context"
	"fmt"
)

// CanvasDocumentComment holds the document_comment subfields of a canvas message.
type CanvasDocumentComment struct {
	ThreadID string   `json:"thread_id"`
	Authors  []string `json:"authors"`
}

// CanvasMessage is a thread root message returned by conversations.history for a canvas channel.
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
