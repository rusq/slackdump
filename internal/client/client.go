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
	"net/http"

	"github.com/rusq/chttp/v2"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/internal/edge"
	"github.com/rusq/slackdump/v4/internal/network"
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
	GetUserInfoContext(ctx context.Context, user string) (*slack.User, error)
	GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error)
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)
	GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination
	ListBookmarks(channelID string) ([]slack.Bookmark, error)
	SearchFilesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchFiles, error)
	SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error)
	GetUserProfileContext(ctx context.Context, params *slack.GetUserProfileParameters) (*slack.UserProfile, error)
}

// CanvasRootClient is an interface that Clients that support Canvas Thread
// Root extraction implement.
type CanvasRootClient interface {
	// CanvasSupported should return true if the client supports extracting
	// canvas threads.
	CanvasSupported() bool
	CanvasThreadRoots(context.Context, string) ([]slack.Message, error)
}

// ErrOpNotSupported is returned when a client lacks an optional Edge or
// canvas capability.
var ErrOpNotSupported = errors.New("client doesn't support this operation")

// ErrSecureFileDownloadUnavailable is returned when a wrapped client has no
// HTTP client whose redirect policy Slackdump controls.
var ErrSecureFileDownloadUnavailable = errors.New("secure file download is unavailable")

var _ Slack = (*Client)(nil)

// Client wraps *slack.Client and, for xoxc sessions, a canvas-capable web
// client. Enterprise workspaces also use that web client for conversation
// overrides exposed through Edge.
type Client struct {
	*slack.Client              // always set; promotes all Slack API methods
	edge          *edge.Client // nil for non-enterprise workspaces
	canvas        *edge.Client // nil unless initialised with an xoxc token
	wi            *slack.AuthTestResponse
	hcl           *http.Client
	fileClient    *slack.Client
}

// Wrap wraps a *slack.Client and returns a *Client that implements the Slack
// interface. Intended for testing. The wrapper does not take ownership of the
// caller's HTTP client, so Close is a no-op for wrapped clients.
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
func newSlackClient(ctx context.Context, prov auth.Provider) (*http.Client, *slack.Client, *slack.AuthTestResponse, error) {
	cl, err := prov.HTTPClient()
	if err != nil {
		return nil, nil, nil, err
	}
	scl := slack.New(prov.SlackToken(), slack.OptionHTTPClient(cl))
	wi, err := scl.AuthTestContext(ctx)
	if err != nil {
		return nil, nil, nil, errors.Join(err, chttp.Close(cl))
	}
	return cl, scl, wi, nil
}

// newFileClient creates a Slack client for file downloads only. It shares the
// authenticated transport and cookie jar with hcl but rejects redirects to a
// URL that is not a trusted Slack file URL.
func newFileClient(token string, hcl *http.Client) *slack.Client {
	fileHTTPClient := *hcl
	priorCheckRedirect := fileHTTPClient.CheckRedirect
	fileHTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := network.ValidateSlackFileURL(req.URL.String()); err != nil {
			return err
		}
		if priorCheckRedirect != nil {
			return priorCheckRedirect(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return slack.New(token, slack.OptionHTTPClient(&fileHTTPClient))
}

// New creates a Client and runs auth.test. Client-token sessions receive a
// canvas-capable web client; Enterprise sessions additionally use it for
// conversation API overrides.
func New(ctx context.Context, prov auth.Provider, opts ...Option) (*Client, error) {
	hcl, scl, wi, err := newSlackClient(ctx, prov)
	if err != nil {
		return nil, err
	}

	var opt options
	for _, o := range opts {
		o(&opt)
	}

	c := &Client{
		Client:     scl,
		wi:         wi,
		hcl:        hcl,
		fileClient: newFileClient(prov.SlackToken(), hcl),
	}

	if auth.IsClientToken(prov.SlackToken()) {
		ecl, err := edge.NewWithInfo(wi, prov)
		if err != nil {
			return nil, errors.Join(err, chttp.Close(hcl))
		}
		c.canvas = ecl
		if opt.enterprise || wi.EnterpriseID != "" {
			c.edge = ecl
		}
	}
	return c, nil
}

// GetFileContext retrieves a private Slack file after validating its URL. The
// dedicated file client also validates every redirect before issuing it.
func (c *Client) GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error {
	if err := network.ValidateSlackFileURL(downloadURL); err != nil {
		return err
	}
	if c.fileClient == nil {
		return ErrSecureFileDownloadUnavailable
	}
	return c.fileClient.GetFileContext(ctx, downloadURL, writer)
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

// Close releases any HTTP transports owned by the client.
func (c *Client) Close() error {
	var err error
	if c.hcl != nil {
		err = errors.Join(err, chttp.Close(c.hcl))
	}
	if c.edge != nil {
		err = errors.Join(err, c.edge.Close())
	}
	if c.canvas != nil && c.canvas != c.edge {
		err = errors.Join(err, c.canvas.Close())
	}
	return err
}

// CanvasSupported reports whether this client was initialised with the xoxc
// client token required by Slack's canvas endpoints.
func (c *Client) CanvasSupported() bool {
	return c.canvas != nil
}

// CanvasThreadRoots returns complete Slack root messages for canvas comment
// threads. Canvas APIs are deliberately unavailable to bot, app, and other
// non-xoxc sessions.
func (c *Client) CanvasThreadRoots(ctx context.Context, fileID string) ([]slack.Message, error) {
	if c.canvas == nil {
		return nil, ErrOpNotSupported
	}
	roots, err := c.canvas.CanvasThreadRoots(ctx, fileID)
	if err != nil {
		return nil, err
	}
	messages := make([]slack.Message, len(roots))
	for i := range roots {
		messages[i] = roots[i].Message
	}
	return messages, nil
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
