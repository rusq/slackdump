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
	"errors"
	"log/slog"
	"runtime/trace"

	"github.com/google/uuid"
	"github.com/rusq/slack"
)

// search.* API

const perPage = 100

var ErrPagination = errors.New("pagination fault")

type SearchResponse[T any] struct {
	baseResponse
	Module     string          `json:"module"`
	Query      string          `json:"query"`
	Filters    json.RawMessage `json:"filters"`
	Pagination Pagination      `json:"pagination"`
	Items      []T             `json:"items"`
}

// searchForm is the form to be sent to the search endpoint.
type searchForm struct {
	BaseRequest
	Cursor               string            `json:"cursor,omitempty"`
	Module               string            `json:"module"`
	Query                string            `json:"query"`
	Page                 int               `json:"page,omitempty"`
	ClientReqID          string            `json:"client_req_id"`
	BrowseID             string            `json:"browse_session_id"`
	Extracts             int               `json:"extracts"`
	Highlight            int               `json:"highlight"`
	ExtraMsg             int               `json:"extra_message_data"`
	NoUserProfile        int               `json:"no_user_profile"`
	Count                int               `json:"count"`
	FileTitleOnly        bool              `json:"file_title_only"`
	QueryRewriteDisabled bool              `json:"query_rewrite_disabled"`
	IncludeFilesShares   int               `json:"include_files_shares"`
	Browse               string            `json:"browse"`
	SearchContext        string            `json:"search_context"`
	MaxFilterSuggestions int               `json:"max_filter_suggestions"`
	Sort                 searchSortType    `json:"sort"`
	SortDir              searchSortDir     `json:"sort_dir"`
	ChannelType          SearchChannelType `json:"channel_type"`
	ExcludeMyChannels    int               `json:"exclude_my_channels"`
	SearchOnlyMyChannels bool              `json:"search_only_my_channels"`
	RecommendSource      string            `json:"recommend_source"`
	WebClientFields
}

// SearchChannelType is the type of the channel to search for SearchChannels
// call.
type SearchChannelType string

const (
	SCTPrivate         SearchChannelType = "private"
	SCTArchived        SearchChannelType = "archived"
	SCTExternalShared  SearchChannelType = "external_shared"
	SCTExcludeArchived SearchChannelType = "exclude_archived"
	SCTPrivateExclude  SearchChannelType = "private_exclude"
	SCTAll             SearchChannelType = ""
)

type searchSortDir string

const (
	ssdEmpty searchSortDir = ""
	ssdAsc   searchSortDir = "asc"
	ssdDesc  searchSortDir = "desc"
)

type searchSortType string

const (
	sstRecommended searchSortType = "recommended"
	sstName        searchSortType = "name"
)

type SearchChannelsParameters struct {
	// OnlyMyChannels instructs the search to return only channels that the
	// current user is a participant of.
	OnlyMyChannels bool
	// ChannelTypes restricts the channel results to a certain type.
	ChannelTypes SearchChannelType
}

// attr returns log attributes.
func (p *SearchChannelsParameters) attr() slog.Attr {
	return slog.GroupAttrs("search_channels_parameters", slog.String("channel_types", string(p.ChannelTypes)), slog.Bool("only_my_channels", p.OnlyMyChannels))
}

const defNumChannels = 5 // initial channel slice size, allocates space for #random, #everyone, @user + a couple of custom

func (cl *Client) SearchChannels(ctx context.Context, query string, p SearchChannelsParameters) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "SearchChannels")
	defer task.End()
	lg := slog.With("in", "SearchChannels", "query", query, p.attr())

	trace.Logf(ctx, "params", "query=%q", query)

	clientReq, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	browseID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	form := searchForm{
		BaseRequest:          BaseRequest{Token: cl.token},
		Module:               "channels",
		Query:                query,
		Page:                 0,
		ClientReqID:          clientReq.String(),
		BrowseID:             browseID.String(),
		Extracts:             0,
		Highlight:            iFalse,
		Cursor:               "*",
		ExtraMsg:             0,
		NoUserProfile:        iTrue,
		Count:                perPage,
		FileTitleOnly:        false,
		QueryRewriteDisabled: false,
		IncludeFilesShares:   iTrue,
		Browse:               "standard",
		SearchContext:        "desktop_channel_browser",
		MaxFilterSuggestions: 10,
		Sort:                 sstName,
		SortDir:              ssdAsc,
		ChannelType:          p.ChannelTypes,
		ExcludeMyChannels:    iFalse,
		SearchOnlyMyChannels: p.OnlyMyChannels,
		RecommendSource:      "channel-browser",
		WebClientFields: WebClientFields{
			XReason:  "browser-query",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
	}

	const ep = "search.modules.channels"
	lim := tier2boost.limiter()
	var cc = make([]slack.Channel, 0, defNumChannels)
	for {
		resp, err := cl.PostForm(ctx, ep, values(form, true))
		if err != nil {
			return nil, err
		}
		var sr SearchResponse[slack.Channel]
		if err := cl.ParseResponse(&sr, resp); err != nil {
			return nil, err
		}
		if err := sr.validate(ep); err != nil {
			return nil, err
		}
		cc = append(cc, sr.Items...)
		if sr.Pagination.NextCursor == "" {
			lg.Debug("no more channels")
			break
		}
		lg.DebugContext(ctx, "pagination", "next_cursor", sr.Pagination.NextCursor)
		form.Cursor = sr.Pagination.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	trace.Logf(ctx, "info", "channels found=%d", len(cc))
	lg.DebugContext(ctx, "channels", "count", len(cc))
	return cc, nil
}
