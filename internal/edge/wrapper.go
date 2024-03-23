package edge

import (
	"context"
	"errors"

	"github.com/rusq/slack"
)

var ErrParameterMissing = errors.New("required parameter missing")

// High level functions that wrap low level calls to webclient API to return
// the data in the format close to the  Slack API.

func (cl *Client) GetConversationsContext(ctx context.Context, _ *slack.GetConversationsParameters) (channels []slack.Channel, _ string, err error) {
	ch, err := cl.SearchChannels(ctx, "")
	if err != nil {
		return nil, "", err
	}
	ims, err := cl.IMList(ctx)
	if err != nil {
		return nil, "", err
	}
	for _, c := range ims {
		ch = append(ch, c.SlackChannel())
	}
	cr, err := cl.ClientCounts(ctx)
	if err != nil {
		return nil, "", err
	}
	for _, c := range cr.MPIMs {
		ch = append(ch, slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:     c.ID,
					IsMpIM: true,
				},
			},
		})
	}
	return ch, "", nil
}

func (cl *Client) GetUsersInConversationContext(ctx context.Context, p *slack.GetUsersInConversationParameters) (ids []string, _ string, err error) {
	if p.ChannelID == "" {
		return nil, "", ErrParameterMissing

	}
	var channelIDs []string
	if p.ChannelID != "" {
		channelIDs = append(channelIDs, p.ChannelID)
	}
	uu, err := cl.UsersList(ctx, channelIDs)
	if err != nil {
		return nil, "", err
	}
	for _, u := range uu {
		ids = append(ids, u.ID)
	}
	return ids, "", nil
}
