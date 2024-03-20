package edge

import (
	"context"
	"log/slog"
	"net/url"
	"time"
)

type dmsForm struct {
	Token          string `json:"token"`
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
	Channel Channel `json:"channel,omitempty"`
	Latest  string  `json:"latest,omitempty"` // i.e. "1710632873.037269"
}

type Channel struct {
	ID            string `json:"id"`
	Created       int64  `json:"created"`
	IsFrozen      bool   `json:"is_frozen"`
	IsArchived    bool   `json:"is_archived"`
	IsIM          bool   `json:"is_im"`
	IsOrgShared   bool   `json:"is_org_shared"`
	ContextTeamID string `json:"context_team_id"`
	Updated       int64  `json:"updated"`
	User          string `json:"user"`
	LastRead      string `json:"last_read"`
	Latest        string `json:"latest"`
	IsOpen        bool   `json:"is_open"`
}

func (cl *Client) DMs(ctx context.Context) ([]DM, error) {
	form := dmsForm{
		Token:          cl.token,
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
	var url = cl.webapiURL("client.dms")
	slog.Info("url", "url", url)
	for range 3 {
		resp, err := cl.PostFormRaw(ctx, url, form.Values())
		if err != nil {
			return nil, err
		}
		r := dmsResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		IMs = append(IMs, r.IMs...)
		time.Sleep(300 * time.Millisecond) //TODO: hax
		form.Cursor = r.ResponseMetadata.NextCursor
	}
	return IMs, nil
}
