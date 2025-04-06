package client

import (
	"context"
	"io"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/edge"
)

// Slack is an interface that defines the methods that a Slack client should provide.
type Slack interface {
	AuthTestContext(ctx context.Context) (response *slack.AuthTestResponse, err error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error)
	GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error)
	GetEmojiContext(ctx context.Context) (map[string]string, error)
	GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error
	GetFileInfoContext(ctx context.Context, fileID string, count int, page int) (*slack.File, []slack.Comment, *slack.Paging, error)
	GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error)
	GetUserInfoContext(ctx context.Context, user string) (*slack.User, error)
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)
	GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination
	ListBookmarks(channelID string) ([]slack.Bookmark, error)
	SearchFilesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchFiles, error)
	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
}

//go:generate mockgen -destination mock_client/mock_client.go . SlackClienter,Slack
type SlackClienter interface {
	Slack
	Client() *slack.Client
}

type Client struct {
	Slack
	wi *slack.AuthTestResponse
}

// Wrap wraps a Slack client and returns a Client that implements the
// SlackClienter interface. This is useful for testing purposes.
func Wrap(cl *slack.Client) *Client {
	return &Client{
		Slack: cl,
	}
}

type options struct {
	enterprise bool
}

type Option func(*options)

func WithEnterprise(enterprise bool) Option {
	return func(o *options) {
		o.enterprise = enterprise
	}
}

func New(ctx context.Context, prov auth.Provider, opts ...Option) (*Client, error) {
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

	client := &Client{
		Slack: scl,
		wi:    wi,
	}

	if opt.enterprise || wi.EnterpriseID != "" {
		ecl, err := edge.NewWithInfo(wi, prov)
		if err != nil {
			return nil, err
		}
		client.Slack = &edgeClient{
			Slack: scl,
			edge:  ecl,
		}
	}
	return client, nil
}

// AuthTestContext returns the cached workspace information that was captured
// on initialisation.
func (c *Client) AuthTestContext(ctx context.Context) (response *slack.AuthTestResponse, err error) {
	if c.wi == nil {
		wi, err := c.Slack.AuthTestContext(ctx)
		if err != nil {
			return nil, err
		}
		c.wi = wi
	}
	return c.wi, nil
}

func (c *Client) Client() *slack.Client {
	switch t := c.Slack.(type) {
	case *edgeClient:
		return t.Slack.(*slack.Client)
	case *slack.Client:
		return t
	default:
		panic("unknown client type")
	}
}

// edgeClient is a wrapper around the edge client that implements the
// Slack interface.  It overrides the methods that don't work on
// enterprise workspaces.
type edgeClient struct {
	Slack
	edge *edge.Client
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
