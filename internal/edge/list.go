package edge

import (
	"context"
	"iter"
	"runtime/trace"
)

// list.* API

type SavedListFilter string

const (
	SavedListFilterSaved     SavedListFilter = "saved"
	SavedListFilterArchived  SavedListFilter = "archived"
	SavedListFilterCompleted SavedListFilter = "completed"
)

type SavedListRequest struct {
	BaseRequest
	Limit             int64           `json:"limit"`
	IncludeTombstones bool            `json:"include_tombstones"`
	Filter            SavedListFilter `json:"filter"` // "saved","archived","completed"
	Cursor            string          `json:"cursor"`
	WebClientFields
}

// SavedListResopnse is the responce for saved.list API endpoint.
type SavedListResponse struct {
	baseResponse
	SavedItems []SavedListItem `json:"saved_items"`
	Counts     SavedListCounts `json:"counts"`
}

type SavedListCounts struct {
	UncompletedCount        int64 `json:"uncompleted_count"`
	UncompletedOverdueCount int64 `json:"uncompleted_overdue_count"`
	ArchivedCount           int64 `json:"archived_count"`
	CompletedCount          int64 `json:"completed_count"`
	TotalCount              int64 `json:"total_count"`
}

type SavedListItem struct {
	ItemID           string  `json:"item_id"`
	ItemType         string  `json:"item_type"`
	DateCreated      int64   `json:"date_created"`
	DateDue          int64   `json:"date_due"`
	DateCompleted    int64   `json:"date_completed"`
	DateUpdated      int64   `json:"date_updated"`
	IsArchived       bool    `json:"is_archived"`
	DateSnoozedUntil int64   `json:"date_snoozed_until"`
	Ts               *string `json:"ts,omitempty"`
	State            string  `json:"state"`
}

func (cl *Client) SavedList(ctx context.Context, filter SavedListFilter) iter.Seq2[SavedListResponse, error] {
	var cursor string
	return func(yield func(SavedListResponse, error) bool) {
		ctx, task := trace.NewTask(ctx, "edge.SavedList")
		defer task.End()
		for {
			resp, err := cl.savedListPage(ctx, filter, cursor)
			if !yield(resp, err) {
				return
			}
			if resp.ResponseMetadata.NextCursor == "" {
				break
			}
			cursor = resp.ResponseMetadata.NextCursor
		}
	}
}

func (cl *Client) savedListPage(ctx context.Context, filter SavedListFilter, cursor string) (SavedListResponse, error) {
	const defLimit = 15
	form := &SavedListRequest{
		BaseRequest:       BaseRequest{Token: cl.token},
		Limit:             defLimit,
		IncludeTombstones: true,
		Filter:            filter,
		Cursor:            cursor,
		WebClientFields:   webclientReason("saved-api/savedList"),
	}
	var resp SavedListResponse
	// original url also has "?slack_route=<TEAM_ID>"
	hr, err := cl.PostForm(ctx, "saved.list", values(form, true))
	if err != nil {
		return SavedListResponse{}, err
	}
	if err := cl.ParseResponse(&resp, hr); err != nil {
		return SavedListResponse{}, err
	}
	return resp, nil
}
