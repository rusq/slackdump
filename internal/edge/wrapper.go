package edge

import (
	"context"
	"io"

	"github.com/rusq/slack"
)

type Wrapper struct {
	cl   *slack.Client
	edge *Client
}

// NewWrapper wraps the slack.Client with the edge client, so that the edge
// client can be used as a fallback.
func (cl *Client) NewWrapper(scl *slack.Client) *Wrapper {
	return &Wrapper{cl: scl, edge: cl}
}

func (w *Wrapper) SlackClient() *slack.Client {
	return w.cl
}

func (w *Wrapper) EdgeClient() *Client {
	return w.edge
}

func (w *Wrapper) AuthTestContext(ctx context.Context) (response *slack.AuthTestResponse, err error) {
	return w.cl.AuthTestContext(ctx)
}

func (w *Wrapper) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	return w.cl.GetConversationHistoryContext(ctx, params)
}
func (w *Wrapper) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	return w.cl.GetConversationRepliesContext(ctx, params)
}
func (w *Wrapper) GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination {
	return w.cl.GetUsersPaginated(options...)
}

func (w *Wrapper) GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error) {
	return w.cl.GetStarredContext(ctx, params)
}

func (w *Wrapper) ListBookmarks(channelID string) ([]slack.Bookmark, error) {
	return w.cl.ListBookmarks(channelID)
}

func (w *Wrapper) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	return w.edge.GetConversationsContext(ctx, params)
}

func (w *Wrapper) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	return w.edge.GetConversationInfoContext(ctx, input)
}

func (w *Wrapper) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	return w.edge.GetUsersInConversationContext(ctx, params)
}

func (w *Wrapper) GetFile(downloadURL string, writer io.Writer) error {
	return w.cl.GetFile(downloadURL, writer)
}

func (w *Wrapper) GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error) {
	return w.cl.GetUsersContext(ctx, options...)
}

func (w *Wrapper) GetEmojiContext(ctx context.Context) (map[string]string, error) {
	return w.cl.GetEmojiContext(ctx)
}
