package edge

import (
	"context"
	"iter"
	"runtime/trace"
)

type emojiResponse struct {
	BaseResponse
	EmojiResult
	CustomEmojiTotalCount int64  `json:"custom_emoji_total_count"`
	Paging                Paging `json:"paging"`
}

type EmojiResult struct {
	Emoji         []Emoji `json:"emoji"`
	DisabledEmoji []Emoji `json:"disabled_emoji,omitempty"`
	Total         int
}

type Emoji struct {
	Name            string   `json:"name"`
	IsAlias         int64    `json:"is_alias,omitempty"`
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
			var r = emojiResponse{
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
			req.Paging.nextPage()
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
		req.Paging.nextPage()
		if err := l.Wait(ctx); err != nil {
			return res, err
		}
	}
	return res, nil
}
