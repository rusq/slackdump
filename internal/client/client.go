// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package client

import (
	"context"
	"io"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/edge"
)

//go:generate mockgen -destination mock_client/mock_client.go . SlackClienter,Slack,SlackEdge

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
	GetUserProfileContext(ctx context.Context, params *slack.GetUserProfileParameters) (*slack.UserProfile, error)
}

// SlackClienter is an extended interface that includes Client method that
// returns the underlying [slack.Client] instance.
type SlackClienter interface {
	Slack
	Client() (*slack.Client, bool)
}

// SlackEdge is an extended interface that includes Edge methods.
type SlackEdge interface {
	SlackClienter
	Edge() (*edge.Client, bool)
	// TODO: additional methods from edge client.
}

var (
	_ Slack         = (*Client)(nil)
	_ SlackClienter = (*Client)(nil)
)

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

// WithEnterprise sets the enterprise flag.  Setting the flag to true forces
// the use of the edge client, even if the workspace is not an enterprise
// workspace.
func WithEnterprise(enterprise bool) Option {
	return func(o *options) {
		o.enterprise = enterprise
	}
}

// New creates a new Client instance.  It checks if workspace provider is
// valid, and checks if it's an enterprise workspace.  If it is, it creates an
// edge client.
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

func (c *Client) Client() (*slack.Client, bool) {
	switch t := c.Slack.(type) {
	case *edgeClient:
		return t.Slack.(*slack.Client), true
	case *slack.Client:
		return t, true
	default:
		return nil, false
	}
}
