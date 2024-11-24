package edge

import (
	"context"

	"github.com/rusq/slack"
)

// files.* API

type filesListForm struct {
	BaseRequest
	Channel string `json:"channel"`
	Count   int    `json:"count"`
	Page    int    `json:"page"`
	WebClientFields
}

type filesListResponse struct {
	baseResponse
	Files []slack.File `json:"files"`
	Pagination
}

func (cl *Client) FilesList(ctx context.Context, channel string, count int) ([]slack.File, error) {
	form := filesListForm{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channel,
		Count:           count,
		WebClientFields: webclientReason("about-modal/sharedFiles"),
	}
	lim := tier3.limiter()
	var ff []slack.File
	for {
		resp, err := cl.Post(ctx, "files.list", form)
		if err != nil {
			return nil, err
		}
		r := filesListResponse{}
		if err := cl.ParseResponse(&r, resp); err != nil {
			return nil, err
		}
		ff = append(ff, r.Files...)
		if form.Page == int(r.Pagination.PageCount) || r.Pagination.PageCount == 0 {
			break
		}
		form.Page++
		if err := lim.Wait(ctx); err != nil {
			return nil, err
		}
	}
	return ff, nil
}
