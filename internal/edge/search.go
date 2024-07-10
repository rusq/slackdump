package edge

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/trace"

	"github.com/google/uuid"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/logger"
)

// search.* API

const perPage = 100

var (
	ErrPagination = errors.New("pagination fault")
)

type SearchResponse[T any] struct {
	BaseResponse
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

func (cl *Client) SearchChannels(ctx context.Context, query string) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "SearchChannels")
	defer task.End()
	trace.Logf(ctx, "params", "query=%q", query)

	lg := logger.FromContext(ctx)
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
		Highlight:            0,
		Cursor:               "*",
		ExtraMsg:             0,
		NoUserProfile:        1,
		Count:                perPage,
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
	lim := tier2boost.limiter()
	var cc []slack.Channel
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
		lg.Debugf("next_cursor=%q", sr.Pagination.NextCursor)
		form.Cursor = sr.Pagination.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	trace.Logf(ctx, "info", "channels found=%d", len(cc))
	return cc, nil
}
