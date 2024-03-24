package edge

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/google/uuid"
	"github.com/rusq/slack"
)

// search.* API

type SearchResponse[T any] struct {
	BaseResponse
	Module     string          `json:"module"`
	Query      string          `json:"query"`
	Filters    json.RawMessage `json:"filters"`
	Pagination Pagination      `json:"pagination"`
	Items      []T             `json:"items"`
}

type Pagination struct {
	TotalCount int64 `json:"total_count"`
	Page       int64 `json:"page"`
	PerPage    int64 `json:"per_page"`
	PageCount  int64 `json:"page_count"`
	First      int64 `json:"first"`
	Last       int64 `json:"last"`
}

// searchForm is the form to be sent to the search endpoint.
type searchForm struct {
	BaseRequest
	Module               string            `json:"module"`
	Query                string            `json:"query"`
	Page                 int               `json:"page"`
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
	ChannelType          searchChannelType `json:"channel_type"`
	ExcludeMyChannels    int               `json:"exclude_my_channels"`
	SearchOnlyMyChannels bool              `json:"search_only_my_channels"`
	RecommendSource      string            `json:"recommend_source"`
	WebClientFields
}

type searchChannelType string

const (
	sctPublic          searchChannelType = "public"
	sctPrivate         searchChannelType = "private"
	scpArchived        searchChannelType = "archived"
	scpExternalShared  searchChannelType = "external_shared"
	scpExcludeArchived searchChannelType = "exclude_archived"
	scpPrivateExclude  searchChannelType = "private_exclude"
	scpAll             searchChannelType = ""
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

func (s *searchForm) Values() url.Values {
	return values(s, false)
}

func (cl *Client) SearchChannels(ctx context.Context, query string) ([]slack.Channel, error) {
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
		Page:                 1,
		ClientReqID:          clientReq.String(),
		BrowseID:             browseID.String(),
		Extracts:             0,
		Highlight:            0,
		ExtraMsg:             0,
		NoUserProfile:        1,
		Count:                50,
		FileTitleOnly:        false,
		QueryRewriteDisabled: false,
		IncludeFilesShares:   1,
		Browse:               "standard",
		SearchContext:        "desktop_channel_browser",
		MaxFilterSuggestions: 10,
		Sort:                 sstName,
		SortDir:              ssdAsc,
		ChannelType:          scpAll,
		ExcludeMyChannels:    0,
		SearchOnlyMyChannels: false,
		RecommendSource:      "channel-browser",
		WebClientFields: WebClientFields{
			XReason:  "browser-query",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
	}

	const ep = "search.modules.channels"
	lim := tier2.limiter()
	var cc []slack.Channel
	for {
		resp, err := cl.PostForm(ctx, ep, form.Values())
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
		if form.Page == int(sr.Pagination.PageCount) || sr.Pagination.PageCount == 0 {
			break
		}
		form.Page++
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return cc, nil
}
