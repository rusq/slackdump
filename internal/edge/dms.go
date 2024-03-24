package edge

import (
	"context"
)

// im.* API

type imListForm struct {
	BaseRequest
	GetLatest    bool   `json:"get_latest"`
	GetReadState bool   `json:"get_read_state"`
	Cursor       string `json:"cursor,omitempty"`
	WebClientFields
}

type imListResponse struct {
	BaseResponse
	IMs []IM `json:"ims,omitempty"`
}

func (cl *Client) IMList(ctx context.Context) ([]IM, error) {
	form := imListForm{
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
	lim := tier2.limiter()
	var IMs []IM
	for {
		resp, err := cl.PostForm(ctx, "im.list", values(form, true))
		if err != nil {
			return nil, err
		}
		r := imListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		IMs = append(IMs, r.IMs...)
		if r.ResponseMetadata.NextCursor == "" {
			break
		}
		form.Cursor = r.ResponseMetadata.NextCursor
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return IMs, nil
}
