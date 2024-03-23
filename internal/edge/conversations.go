package edge

import (
	"context"
	"encoding/json"

	"github.com/rusq/slack"
)

// conversations.* API

// conversationsGenericInfoForm is the request to conversations.genericInfo
type conversationsGenericInfoForm struct {
	BaseRequest
	UpdatedChannels string `json:"updated_channels"` // i.e. {"C065H568ZAT":0}
	WebClientFields
}

type conversationsGenericInfoResponse struct {
	BaseResponse
	Channels            []slack.Channel `json:"channels"`
	UnchangedChannelIDs []string        `json:"unchanged_channel_ids"`
}

func (cl *Client) ConversationsGenericInfo(ctx context.Context, channelID ...string) ([]slack.Channel, error) {
	updChannel := make(map[string]int, len(channelID))
	for _, id := range channelID {
		updChannel[id] = 0
	}
	b, err := json.Marshal(updChannel)
	if err != nil {
		return nil, err
	}
	form := conversationsGenericInfoForm{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		UpdatedChannels: string(b),
		WebClientFields: WebClientFields{
			XReason:  "fallback:UnknownFetchManager",
			XMode:    "online",
			XSonic:   true,
			XAppName: "client",
		},
	}
	resp, err := cl.PostForm(ctx, "conversations.genericInfo", values(form, true))
	if err != nil {
		return nil, err
	}
	var r conversationsGenericInfoResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, err
	}
	return r.Channels, nil
}
