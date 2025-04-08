package client

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/edge"
)

var (
	_ Slack     = (*edgeClient)(nil)
	_ SlackEdge = (*edgeClient)(nil)
)

// edgeClient is a wrapper around the edge client that implements the
// Slack interface.  It overrides the methods that don't work on
// enterprise workspaces.
type edgeClient struct {
	Slack
	edge *edge.Client
}

func (w *edgeClient) Client() (*slack.Client, bool) {
	switch t := w.Slack.(type) {
	case *slack.Client:
		return t, true
	}
	return nil, false
}

func (w *edgeClient) Edge() (*edge.Client, bool) {
	return w.edge, true
}

func (w *edgeClient) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	return w.edge.GetConversationsContext(ctx, params)
}

func (w *edgeClient) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	return w.edge.GetConversationInfoContext(ctx, input)
}

func (w *edgeClient) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	return w.edge.GetUsersInConversationContext(ctx, params)
}
