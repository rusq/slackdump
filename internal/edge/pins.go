package edge

import (
	"context"

	"github.com/rusq/slack"
)

// pins.* API
type pinsListRequest struct {
	BaseRequest
	Channel string `json:"channel"`
	WebClientFields
}

type pinsListResponse struct {
	baseResponse
	Items []PinnedItem `json:"items"`
}

type PinnedItem struct {
	Type      string        `json:"type"`
	Created   int64         `json:"created"`
	CreatedBy string        `json:"created_by"`
	Channel   string        `json:"channel"`
	Message   slack.Message `json:"message"`
}

// PinsList resturns a list of pinned items in a conversation.
func (cl *Client) PinsList(ctx context.Context, channelID string) ([]PinnedItem, error) {
	form := &pinsListRequest{
		BaseRequest:     BaseRequest{Token: cl.token},
		Channel:         channelID,
		WebClientFields: webclientReason("fetchPinsList"),
	}
	var resp pinsListResponse
	hr, err := cl.PostForm(ctx, "pins.list", values(form, true))
	if err != nil {
		return nil, err
	}
	if err := cl.ParseResponse(&resp, hr); err != nil {
		return nil, err
	}
	return resp.Items, nil
}
