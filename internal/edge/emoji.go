package edge

import "context"

type EmojiResponse struct {
	BaseResponse
	EmojiResult
	CustomEmojiTotalCount int64  `json:"custom_emoji_total_count"`
	Paging                Paging `json:"paging"`
}

type EmojiResult struct {
	Emoji         []Emoji `json:"emoji"`
	DisabledEmoji []Emoji `json:"disabled_emoji,omitempty"`
}

type Emoji struct {
	Name            string   `json:"name"`
	IsAlias         int64    `json:"is_alias"`
	AliasFor        string   `json:"alias_for"`
	URL             string   `json:"url"`
	TeamID          string   `json:"team_id"`
	UserID          string   `json:"user_id"`
	Created         int64    `json:"created"`
	IsBad           bool     `json:"is_bad"`
	UserDisplayName string   `json:"user_display_name"`
	AvatarHash      string   `json:"avatar_hash"`
	CanDelete       bool     `json:"can_delete"`
	Synonyms        []string `json:"synonyms"`
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

// AdminEmojiList returns a list of custom emoji for the workspace.
func (cl *Client) AdminEmojiList(ctx context.Context) (EmojiResult, error) {
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
		var r EmojiResponse
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
