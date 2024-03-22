package edge

import (
	"context"
	"net/url"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fasttime"
)

type dmsForm struct {
	BaseRequest
	Count          int    `json:"count"`
	IncludeClosed  bool   `json:"include_closed"`
	IncludeChannel bool   `json:"include_channel"`
	ExcludeBots    bool   `json:"exclude_bots"`
	Cursor         string `json:"cursor,omitempty"`
	WebClientFields
}

func (d dmsForm) Values() url.Values {
	return values(d, true)
}

type dmsResponse struct {
	BaseResponse
	IMs   []DM `json:"ims,omitempty"`
	MPIMs []DM `json:"mpims,omitempty"` //TODO
}

type DM struct {
	ID string `json:"id"`
	// Message slack.Message `json:"message,omitempty"`
	Channel Channel       `json:"channel,omitempty"`
	Latest  fasttime.Time `json:"latest,omitempty"` // i.e. "1710632873.037269"
}

type Channel struct {
	ID            string         `json:"id"`
	Created       slack.JSONTime `json:"created"`
	IsFrozen      bool           `json:"is_frozen"`
	IsArchived    bool           `json:"is_archived"`
	IsIM          bool           `json:"is_im"`
	IsOrgShared   bool           `json:"is_org_shared"`
	ContextTeamID string         `json:"context_team_id"`
	Updated       slack.JSONTime `json:"updated"`
	User          string         `json:"user"`
	LastRead      fasttime.Time  `json:"last_read"`
	Latest        fasttime.Time  `json:"latest"`
	IsOpen        bool           `json:"is_open"`
}

func (cl *Client) DMs(ctx context.Context) ([]DM, error) {
	form := dmsForm{
		BaseRequest:    BaseRequest{Token: cl.token},
		Count:          250,
		IncludeClosed:  true,
		IncludeChannel: true,
		ExcludeBots:    false,
		Cursor:         "",
		WebClientFields: WebClientFields{
			XReason:  "dms-tab-populate",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
	}

	var IMs []DM
	for {
		resp, err := cl.PostFormRaw(ctx, cl.webapiURL("client.dms"), form.Values())
		if err != nil {
			return nil, err
		}
		r := dmsResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		IMs = append(IMs, r.IMs...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		time.Sleep(300 * time.Millisecond) //TODO: hax
		form.Cursor = r.ResponseMetadata.NextCursor
	}
	return IMs, nil
}

type imsForm struct {
	BaseRequest
	GetLatest    bool   `json:"get_latest"`
	GetReadState bool   `json:"get_read_state"`
	Cursor       string `json:"cursor,omitempty"`
	WebClientFields
}

type imsResponse struct {
	BaseResponse
	IMs []Channel `json:"ims,omitempty"`
}

func (cl *Client) IMs(ctx context.Context) ([]Channel, error) {
	form := imsForm{
		BaseRequest:  BaseRequest{Token: cl.token},
		GetLatest:    true,
		GetReadState: true,
		WebClientFields: WebClientFields{
			XReason:  "guided-search-people-empty-state",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
		Cursor: "",
	}

	var IMs []Channel
	for {
		resp, err := cl.PostFormRaw(ctx, cl.webapiURL("im.list"), values(form, true))
		if err != nil {
			return nil, err
		}
		r := imsResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		IMs = append(IMs, r.IMs...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		time.Sleep(300 * time.Millisecond) //TODO: hax
		form.Cursor = r.ResponseMetadata.NextCursor
	}
	return IMs, nil
}

type countsForm struct {
	BaseRequest
	ThreadCountsByChannel bool `json:"thread_counts_by_channel"`
	OrgWideAware          bool `json:"org_wide_aware"`
	IncludeFileChannels   bool `json:"include_file_channels"`
	WebClientFields
}

type CountsResponse struct {
	BaseResponse
	Channels []ChannelSnapshot `json:"channels,omitempty"`
	MPIMs    []ChannelSnapshot `json:"mpims,omitempty"`
	IMs      []ChannelSnapshot `json:"ims,omitempty"`
}

type ChannelSnapshot struct {
	ID             string        `json:"id"`
	LastRead       fasttime.Time `json:"last_read"`
	Latest         fasttime.Time `json:"latest"`
	HistoryInvalid fasttime.Time `json:"history_invalid"`
	MentionCount   int           `json:"mention_count"`
	HasUnreads     bool          `json:"has_unreads"`
}

func (cl *Client) Counts(ctx context.Context) (CountsResponse, error) {
	form := countsForm{
		BaseRequest:           BaseRequest{Token: cl.token},
		ThreadCountsByChannel: true,
		OrgWideAware:          true,
		IncludeFileChannels:   true,
		WebClientFields: WebClientFields{
			XReason:  "client-counts-api/fetchClientCounts",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
	}

	resp, err := cl.PostForm(ctx, "client.counts", values(form, true))
	if err != nil {
		return CountsResponse{}, err
	}
	r := CountsResponse{}
	if err := cl.ParseResponse(&r, resp); err != nil {
		return CountsResponse{}, err
	}
	return r, nil
}
