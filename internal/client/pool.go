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
	"log/slog"
	"sync"

	"github.com/rusq/slack"
)

// Pool is a pool of Slack clients that can be used to make API calls.
// Zero value is not usable, must be initialised with [NewPool].
type Pool struct {
	pool []Slack
	mu   sync.Mutex
	strategy
}

// NewPool wraps the slack.Client with the edge client, so that the edge
// client can be used as a fallback.
func NewPool(scl ...Slack) *Pool {
	return &Pool{
		pool:     scl,
		strategy: newRoundRobin(len(scl)),
	}
}

// next returns the next client in the pool using the current strategy.
func (p *Pool) next() Slack {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pool) == 0 {
		panic("no clients in pool")
	}
	next := p.strategy.next()
	slog.Debug("next client", "index", next)
	return p.pool[next]
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

func (w *Pool) GetUserProfileContext(ctx context.Context, params *slack.GetUserProfileParameters) (*slack.UserProfile, error) {
	return w.next().GetUserProfileContext(ctx, params)
}
