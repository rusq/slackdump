package client

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/edge"
)

var (
	_ Slack     = (*edgeClient)(nil)
	_ SlackEdge = (*edgeClient)(nil)
)

// NewEdge returns a new Slack Edge client.
func NewEdge(ctx context.Context, prov auth.Provider, opts ...Option) (SlackEdge, error) {
	cl, err := prov.HTTPClient()
	if err != nil {
		return nil, err
	}
	scl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(cl))
	wi, err := scl.AuthTestContext(ctx)
	if err != nil {
		return nil, err
	}
	var opt options
	for _, o := range opts {
		o(&opt)
	}
	return newEdge(prov, scl, wi)
}

func newEdge(prov auth.Provider, scl *slack.Client, wi *slack.AuthTestResponse) (*edgeClient, error) {
	ecl, err := edge.NewWithInfo(wi, prov)
	if err != nil {
		return nil, err
	}
	client := &edgeClient{
		Slack: scl,
		edge:  ecl,
	}

	return client, nil
}

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

func (w *edgeClient) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	p1 := edge.ConversationsHistoryParams{
		ChannelID:     params.ChannelID,
		Oldest:        params.Oldest,
		Latest:        params.Latest,
		Cursor:        params.Cursor,
		Limit:         params.Limit,
		IgnoreReplies: true, // Ignore replies by default
		Inclusive:     params.Inclusive,
	}
	chr, err := w.edge.ConversationsHistory(ctx, p1)
	if err != nil {
		return nil, err
	}
	return &slack.GetConversationHistoryResponse{
		SlackResponse: slack.SlackResponse{
			Ok:    chr.Ok,
			Error: chr.Error,
			ResponseMetadata: slack.ResponseMetadata{
				Cursor:   chr.ResponseMetadata.NextCursor,
				Messages: chr.ResponseMetadata.Messages,
			},
		},
		HasMore:  chr.HasMore,
		PinCount: chr.PinCount,
		Latest:   chr.Latest.SlackString(),
		ResponseMetaData: struct {
			NextCursor string "json:\"next_cursor\""
		}{
			NextCursor: chr.ResponseMetadata.NextCursor,
		},
		Messages: chr.Messages,
	}, nil
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

func (w *edgeClient) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	p := edge.ConversationsRepliesParams{
		Channel:   params.ChannelID,
		TS:        params.Timestamp,
		Limit:     params.Limit,
		Oldest:    params.Oldest,
		Latest:    params.Latest,
		Inclusive: params.Inclusive,
	}
	resp, err := w.edge.ConversationsReplies(ctx, p)
	if err != nil {
		return nil, false, "", err
	}
	return resp.Messages, resp.HasMore, resp.ResponseMetadata.NextCursor, nil
}
