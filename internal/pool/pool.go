package pool

import (
	"context"
	"io"
	"sync"

	"github.com/rusq/slack"
)

// SlackClient is an interface that defines the methods that a Slack client
type SlackClient interface {
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

type Pool struct {
	pool []SlackClient
	mu   sync.Mutex
	strategy
}

// strategy is an interface that defines the strategy for selecting the next
// item.
type strategy interface {
	// next returns the next item in the pool.
	next() int
}

// roundRobin implements the round robin strategy.
type roundRobin struct {
	// total is the total number of items in the pool.
	total int
	// i is the current item index.
	i int
}

// newRoundRobin creates a new round robin strategy with the given total number
// of items.
func newRoundRobin(total int) *roundRobin {
	return &roundRobin{total: total}
}

func (r *roundRobin) next() int {
	r.i = (r.i + 1) % r.total
	return r.i
}

// NewWrapper wraps the slack.Client with the edge client, so that the edge
// client can be used as a fallback.
func (p *Pool) NewWrapper(scl ...SlackClient) *Pool {
	return &Pool{
		pool:     scl,
		strategy: newRoundRobin(len(scl)),
	}
}

// next returns the next client in the pool, round robin style.
func (p *Pool) next() SlackClient {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pool) == 0 {
		panic("no clients in pool")
	}
	return p.pool[p.strategy.next()]
}

func (p *Pool) SlackClient() *slack.Client {
	return p.next().(*slack.Client)
}

func (p *Pool) AuthTestContext(ctx context.Context) (response *slack.AuthTestResponse, err error) {
	return p.next().AuthTestContext(ctx)
}

func (p *Pool) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	return p.next().GetConversationHistoryContext(ctx, params)
}

func (p *Pool) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) (msgs []slack.Message, hasMore bool, nextCursor string, err error) {
	return p.next().GetConversationRepliesContext(ctx, params)
}

func (p *Pool) GetUsersPaginated(options ...slack.GetUsersOption) slack.UserPagination {
	return p.next().GetUsersPaginated(options...)
}

func (p *Pool) GetStarredContext(ctx context.Context, params slack.StarsParameters) ([]slack.StarredItem, *slack.Paging, error) {
	return p.next().GetStarredContext(ctx, params)
}

func (p *Pool) ListBookmarks(channelID string) ([]slack.Bookmark, error) {
	return p.next().ListBookmarks(channelID)
}

func (p *Pool) GetConversationsContext(ctx context.Context, params *slack.GetConversationsParameters) (channels []slack.Channel, nextCursor string, err error) {
	return p.next().GetConversationsContext(ctx, params)
}

func (p *Pool) GetConversationInfoContext(ctx context.Context, input *slack.GetConversationInfoInput) (*slack.Channel, error) {
	return p.next().GetConversationInfoContext(ctx, input)
}

func (p *Pool) GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	return p.next().GetUsersInConversationContext(ctx, params)
}

func (p *Pool) GetFileContext(ctx context.Context, downloadURL string, writer io.Writer) error {
	return p.next().GetFileContext(ctx, downloadURL, writer)
}

func (p *Pool) GetUsersContext(ctx context.Context, options ...slack.GetUsersOption) ([]slack.User, error) {
	return p.next().GetUsersContext(ctx, options...)
}

func (p *Pool) GetEmojiContext(ctx context.Context) (map[string]string, error) {
	return p.next().GetEmojiContext(ctx)
}

func (p *Pool) SearchMessagesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	return p.next().SearchMessagesContext(ctx, query, params)
}

func (p *Pool) SearchFilesContext(ctx context.Context, query string, params slack.SearchParameters) (*slack.SearchFiles, error) {
	return p.next().SearchFilesContext(ctx, query, params)
}

func (p *Pool) GetFileInfoContext(ctx context.Context, fileID string, count int, page int) (*slack.File, []slack.Comment, *slack.Paging, error) {
	return p.next().GetFileInfoContext(ctx, fileID, count, page)
}

func (p *Pool) GetUserInfoContext(ctx context.Context, user string) (*slack.User, error) {
	return p.next().GetUserInfoContext(ctx, user)
}
