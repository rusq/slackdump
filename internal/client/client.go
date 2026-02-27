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
	"errors"
	"io"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/edge"
)

//go:generate mockgen -destination mock_client/mock_client.go . Slack

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

// ErrOpNotSupported is returned by edge-only methods when the Client was not
// initialised with an edge (enterprise) connection.
var ErrOpNotSupported = errors.New("client doesn't support this operation")

var _ Slack = (*Client)(nil)

// Client wraps *slack.Client and, optionally, *edge.Client.  The edge client
// is only present for enterprise workspaces.  All Slack interface methods are
// promoted from the embedded *slack.Client; edge-aware methods override them
// when c.edge is set.
type Client struct {
	*slack.Client              // always set; promotes all Slack API methods
	edge          *edge.Client // nil for non-enterprise workspaces
	wi            *slack.AuthTestResponse
}

// Wrap wraps a *slack.Client and returns a *Client that implements the Slack
// interface. Intended for testing.
func Wrap(cl *slack.Client) *Client {
	return &Client{
		Client: cl,
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

// newSlackClient is a shared helper that dials Slack and runs an auth-test.
func newSlackClient(ctx context.Context, prov auth.Provider) (*slack.Client, *slack.AuthTestResponse, error) {
	cl, err := prov.HTTPClient()
	if err != nil {
		return nil, nil, err
	}
	scl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(cl))
	wi, err := scl.AuthTestContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	return scl, wi, nil
}

// New creates a new Client instance.  It checks if workspace provider is
// valid, and checks if it's an enterprise workspace.  If it is, it creates an
// edge client.
func New(ctx context.Context, prov auth.Provider, opts ...Option) (*Client, error) {
	scl, wi, err := newSlackClient(ctx, prov)
	if err != nil {
		return nil, err
	}

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	c := &Client{
		Client: scl,
		wi:     wi,
	}

	if opt.enterprise || wi.EnterpriseID != "" {
		ecl, err := edge.NewWithInfo(wi, prov)
		if err != nil {
			return nil, err
		}
		c.edge = ecl
	}
	return c, nil
}

// AuthTestContext returns the cached workspace information that was captured
// on initialisation.  If the cache is empty it calls the API.
func (c *Client) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	if c.wi == nil {
		wi, err := c.Client.AuthTestContext(ctx)
		if err != nil {
			return nil, err
		}
		c.wi = wi
	}
	return c.wi, nil
}

// Edge returns the underlying *edge.Client, or nil when the workspace is not
// an enterprise workspace.
func (c *Client) Edge() *edge.Client {
	return c.edge
}

// ---------------------------------------------------------------------------
// Edge-aware method overrides
// ---------------------------------------------------------------------------

// GetConversationsContext overrides the standard method with the edge client
// for enterprise workspaces.
func (c *Client) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	if c.edge != nil {
		return c.edge.GetConversationsContext(ctx, params)
	}
	return c.Client.GetConversationsContext(ctx, params)
}

// GetConversationsContextEx is the extended variant that supports the onlyMy
// parameter available in enterprise workspaces.  Returns [ErrOpNotSupported]
// when there is no edge client.
func (c *Client) GetConversationsContextEx(ctx context.Context, params *slack.GetConversationsParameters, onlyMy bool) ([]slack.Channel, string, error) {
	if c.edge == nil {
		return nil, "", ErrOpNotSupported
	}
	return c.edge.GetConversationsContextEx(ctx, params, onlyMy)
}

// GetConversationInfoContext overrides the standard method with the edge client
// for enterprise workspaces.
func (c *Client) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	if c.edge != nil {
		return c.edge.GetConversationInfoContext(ctx, input)
	}
	return c.Client.GetConversationInfoContext(ctx, input)
}

// GetUsersInConversationContext overrides the standard method with the edge
// client for enterprise workspaces.
func (c *Client) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	if c.edge != nil {
		return c.edge.GetUsersInConversationContext(ctx, params)
	}
	return c.Client.GetUsersInConversationContext(ctx, params)
}
