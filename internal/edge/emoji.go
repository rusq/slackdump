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

package edge

import (
	"context"
	"iter"
	"runtime/trace"
)

type emojiResponse struct {
	baseResponse
	EmojiResult
	CustomEmojiTotalCount int64  `json:"custom_emoji_total_count"`
	Paging                Paging `json:"paging"`
}

// EmojiResult is a subset of the response from the emoji.adminList API.
type EmojiResult struct {
	// Emoji is the list of custom emoji.
	Emoji []Emoji `json:"emoji"`
	// DisabledEmoji is the list of disabled custom emoji (supposedly).
	DisabledEmoji []Emoji `json:"disabled_emoji,omitempty"`
	// Total is the total number of custom emoji.
	Total int
}

// Emoji represents a custom emoji as read by the Client API.
type Emoji struct {
	Name            string   `json:"name"`
	IsAlias         int      `json:"is_alias,omitempty"`
	AliasFor        string   `json:"alias_for,omitempty"`
	URL             string   `json:"url"`
	TeamID          string   `json:"team_id,omitempty"`
	UserID          string   `json:"user_id,omitempty"`
	Created         int64    `json:"created,omitempty"`
	IsBad           bool     `json:"is_bad,omitempty"`
	UserDisplayName string   `json:"user_display_name,omitempty"`
	AvatarHash      string   `json:"avatar_hash,omitempty"`
	CanDelete       bool     `json:"can_delete,omitempty"`
	Synonyms        []string `json:"synonyms,omitempty"`
}

type Paging struct {
	Count int64 `json:"count,omitempty"`
	Total int64 `json:"total,omitempty"`
	Page  int64 `json:"page,omitempty"`
	Pages int64 `json:"pages,omitempty"`
}

func (p *Paging) isLastPage() bool {
	return p.Page >= p.Pages || p.Pages == 0
}

func (p *Paging) nextPage() int64 {
	old := p.Page
	p.Page++
	return old
}

type adminEmojiListRequest struct {
	BaseRequest
	WebClientFields
	Paging
}

func (cl *Client) AdminEmojiList(ctx context.Context) iter.Seq2[EmojiResult, error] {
	return func(yield func(EmojiResult, error) bool) {
		ctx, task := trace.NewTask(ctx, "edge.AdminEmojiList")
		defer task.End()

		var res EmojiResult
		req := adminEmojiListRequest{
			BaseRequest: BaseRequest{
				Token: cl.token,
			},
			Paging: Paging{
				Page:  1,
				Count: 100,
			},
			WebClientFields: webclientReason("customize-emoji-new-query"),
		}
		for {
			resp, err := cl.Post(ctx, "emoji.adminList", req)
			if err != nil {
				yield(res, err)
				return
			}
			r := emojiResponse{
				EmojiResult: EmojiResult{
					Emoji: make([]Emoji, 0, 100),
				},
			}
			if err := cl.ParseResponse(&r, resp); err != nil {
				yield(res, err)
				return
			}
			r.Total = int(r.Paging.Total)
			if !yield(r.EmojiResult, nil) {
				return
			}
			if r.Paging.isLastPage() {
				return
			}
			req.nextPage()
		}
	}
}

// AdminEmojiList returns a list of custom emoji for the workspace.
func (cl *Client) AdminEmojiListFull(ctx context.Context) (EmojiResult, error) {
	var res EmojiResult
	req := adminEmojiListRequest{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		Paging: Paging{
			Page:  1,
			Count: 100,
		},
		WebClientFields: webclientReason("customize-emoji-new-query"),
	}
	l := tier2boost.limiter()
	for {
		resp, err := cl.Post(ctx, "emoji.adminList", req)
		if err != nil {
			return res, err
		}
		var r emojiResponse
		if err := cl.ParseResponse(&r, resp); err != nil {
			return res, err
		}
		res.Emoji = append(res.Emoji, r.Emoji...)
		res.DisabledEmoji = append(res.DisabledEmoji, r.DisabledEmoji...)
		if r.Paging.isLastPage() {
			break
		}
		req.nextPage()
		if err := l.Wait(ctx); err != nil {
			return res, err
		}
	}
	return res, nil
}
