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
	"encoding/json"
)

// bookmarks.* API

type bookmarksListForm struct {
	BaseRequest
	Channel                string `json:"channel"`
	IncludeFolders         bool   `json:"include_folders"`
	IncludeLegacyWorkflows bool   `json:"include_legacy_workflows"`
	WebClientFields
}

type bookmarksListResponse struct {
	baseResponse
	Bookmarks []Bookmark `json:"bookmarks"`
}

type Bookmark struct {
	ID                  string          `json:"id"`
	ChannelID           string          `json:"channel_id"`
	Title               json.RawMessage `json:"title"`
	Link                string          `json:"link"`
	Emoji               json.RawMessage `json:"emoji"`
	IconURL             json.RawMessage `json:"icon_url"`
	Type                string          `json:"type"`
	EntityID            json.RawMessage `json:"entity_id"`
	DateCreated         int64           `json:"date_created"`
	DateUpdated         int64           `json:"date_updated"`
	Rank                string          `json:"rank"`
	LastUpdatedByUserID string          `json:"last_updated_by_user_id"`
	LastUpdatedByTeamID string          `json:"last_updated_by_team_id"`
	ShortcutID          string          `json:"shortcut_id"`
	AppID               string          `json:"app_id"`
	AppActionID         string          `json:"app_action_id"`
	ImageURL            json.RawMessage `json:"image_url"`
	DateCreate          int64           `json:"date_create"`
	DateUpdate          int64           `json:"date_update"`
	ParentID            json.RawMessage `json:"parent_id"`
}

// BookmarksList lists bookmarks for a channel.
func (cl *Client) BookmarksList(ctx context.Context, channelID string) ([]Bookmark, error) {
	form := &bookmarksListForm{
		BaseRequest:            BaseRequest{Token: cl.token},
		Channel:                channelID,
		IncludeFolders:         true,
		IncludeLegacyWorkflows: true,
		WebClientFields:        webclientReason("bookmarks-store/conditional-fetching"),
	}
	var resp bookmarksListResponse
	hr, err := cl.PostForm(ctx, "bookmarks.list", values(form, true))
	if err != nil {
		return nil, err
	}
	if err := cl.ParseResponse(&resp, hr); err != nil {
		return nil, err
	}
	return resp.Bookmarks, nil
}
