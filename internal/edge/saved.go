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
	"runtime/trace"
)

// saved.* API, backing Slack's "Later" view (stars.list is deprecated for this, see https://docs.slack.dev/reference/methods/stars.list/).

// SavedItem is a single entry in the current user's "Later" list.
type SavedItem struct {
	ItemID           string `json:"item_id"` // channel ID the item belongs to
	ItemType         string `json:"item_type"`
	Timestamp        string `json:"ts"`
	DateCreated      int64  `json:"date_created"`
	DateDue          int64  `json:"date_due"`
	DateCompleted    int64  `json:"date_completed"`
	DateUpdated      int64  `json:"date_updated"`
	DateSnoozedUntil int64  `json:"date_snoozed_until"`
	IsArchived       bool   `json:"is_archived"`
	State            string `json:"state"`
	TodoState        string `json:"todo_state"`
}

type savedListCounts struct {
	UncompletedCount        int `json:"uncompleted_count"`
	UncompletedOverdueCount int `json:"uncompleted_overdue_count"`
	ArchivedCount           int `json:"archived_count"`
	CompletedCount          int `json:"completed_count"`
	TotalCount              int `json:"total_count"`
}

type savedListForm struct {
	BaseRequest
	Limit             int    `json:"limit"`
	Filter            string `json:"filter"`
	IncludeTombstones bool   `json:"include_tombstones"`
	Cursor            string `json:"cursor,omitempty"`
	WebClientFields
}

type savedListResponse struct {
	baseResponse
	SavedItems []SavedItem     `json:"saved_items"`
	Counts     savedListCounts `json:"counts"`
}

const savedListPageSize = 15 // matches the page size the web client requests

// savedListFilters are the only values the "filter" enum accepts; "all" and
// "" are rejected by the API. Together they cover every item, matching
// counts.total_count.
var savedListFilters = []string{"saved", "completed", "archived"}

// SavedList returns the current user's "Later" items across every state.
func (cl *Client) SavedList(ctx context.Context) ([]SavedItem, error) {
	ctx, task := trace.NewTask(ctx, "SavedList")
	defer task.End()

	var items []SavedItem
	for _, filter := range savedListFilters {
		ii, err := cl.savedListFilter(ctx, filter)
		if err != nil {
			return nil, err
		}
		items = append(items, ii...)
	}
	return items, nil
}

func (cl *Client) savedListFilter(ctx context.Context, filter string) ([]SavedItem, error) {
	form := savedListForm{
		BaseRequest:       BaseRequest{Token: cl.token},
		Limit:             savedListPageSize,
		Filter:            filter,
		IncludeTombstones: true,
		WebClientFields:   webclientReason("saved-api/savedList"),
	}
	lim := tier2boost.limiter()
	var items []SavedItem
	for {
		resp, err := cl.PostForm(ctx, "saved.list", values(form, true))
		if err != nil {
			return nil, err
		}
		r := savedListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		if err := r.validate("saved.list"); err != nil {
			return nil, err
		}
		items = append(items, r.SavedItems...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		form.Cursor = r.ResponseMetadata.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return items, nil
}
