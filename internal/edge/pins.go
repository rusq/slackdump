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

	"github.com/rusq/slack"
)

// pins.* API
type pinsListRequest struct {
	BaseRequest
	Channel string `json:"channel"`
	WebClientFields
}

type pinsListResponse struct {
	baseResponse
	Items []PinnedItem `json:"items"`
}

type PinnedItem struct {
	Type      string        `json:"type"`
	Created   int64         `json:"created"`
	CreatedBy string        `json:"created_by"`
	Channel   string        `json:"channel"`
	Message   slack.Message `json:"message"`
}

// PinsList returns a list of pinned items in a conversation.
func (cl *Client) PinsList(ctx context.Context, channelID string) ([]PinnedItem, error) {
	form := &pinsListRequest{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channelID,
		WebClientFields: webclientReason("fetchPinsList"),
	}
	var resp pinsListResponse
	hr, err := cl.PostForm(ctx, "pins.list", values(form, true))
	if err != nil {
		return nil, err
	}
	if err := cl.ParseResponse(&resp, hr); err != nil {
		return nil, err
	}
	return resp.Items, nil
}
