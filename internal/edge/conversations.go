package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/trace"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
)

// conversations.* API

const safeLimit = 28

// conversationsGenericInfoForm is the request to conversations.genericInfo
type conversationsGenericInfoForm struct {
	BaseRequest
	UpdatedChannels string `json:"updated_channels"` // i.e. {"C065H568ZAT":0}
	WebClientFields
}

type conversationsGenericInfoResponse struct {
	baseResponse
	Channels            []slack.Channel `json:"channels"`
	UnchangedChannelIDs []string        `json:"unchanged_channel_ids"`
}

func (cl *Client) ConversationsGenericInfo(ctx context.Context, channelID ...string) ([]slack.Channel, error) {
	ctx, task := trace.NewTask(ctx, "ConversationsGenericInfo")
	defer task.End()
	trace.Logf(ctx, "params", "channelID=%v", channelID)

	updChannel := make(map[string]int, len(channelID))
	for _, id := range channelID {
		updChannel[id] = 0
	}
	b, err := json.Marshal(updChannel)
	if err != nil {
		return nil, err
	}
	form := conversationsGenericInfoForm{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		UpdatedChannels: string(b),
		WebClientFields: webclientReason("fallback:UnknownFetchManager"),
	}
	resp, err := cl.PostForm(ctx, "conversations.genericInfo", values(form, true))
	if err != nil {
		return nil, err
	}
	var r conversationsGenericInfoResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, err
	}
	return r.Channels, nil
}

type conversationsViewForm struct {
	BaseRequest
	CanonicalAvatars             bool   `json:"canonical_avatars"`
	NoUserProfile                bool   `json:"no_user_profile"`
	IgnoreReplies                bool   `json:"ignore_replies"`
	NoSelf                       bool   `json:"no_self"`
	IncludeFullUsers             bool   `json:"include_full_users"`
	IncludeUseCases              bool   `json:"include_use_cases"`
	IncludeStories               bool   `json:"include_stories"`
	NoMembers                    bool   `json:"no_members"`
	IncludeMutationTimestamps    bool   `json:"include_mutation_timestamps"`
	Count                        int    `json:"count"`
	Channel                      string `json:"channel"`
	IncludeFreeTeamExtraMessages bool   `json:"include_free_team_extra_messages"`
	WebClientFields
}

type ConversationsViewResponse struct {
	Users  []User            `json:"users"`
	IM     IM                `json:"im"`
	Emojis map[string]string `json:"emojis"`
	// we don't care about the rest of the response
}

func (cl *Client) ConversationsView(ctx context.Context, channelID string) (ConversationsViewResponse, error) {
	ctx, task := trace.NewTask(ctx, "ConversationsView")
	defer task.End()
	trace.Logf(ctx, "params", "channelID=%v", channelID)

	form := conversationsViewForm{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		CanonicalAvatars:          true,
		NoUserProfile:             true,
		IgnoreReplies:             true,
		NoSelf:                    true,
		IncludeFullUsers:          false,
		IncludeUseCases:           false,
		IncludeStories:            false,
		NoMembers:                 true,
		IncludeMutationTimestamps: false,
		Count:                     50,
		Channel:                   channelID,
		WebClientFields:           webclientReason(""),
	}
	resp, err := cl.PostForm(ctx, "conversations.view", values(form, true))
	if err != nil {
		return ConversationsViewResponse{}, err
	}
	var r = struct {
		baseResponse
		ConversationsViewResponse
	}{}
	if err := cl.ParseResponse(&r, resp); err != nil {
		return ConversationsViewResponse{}, err
	}
	return r.ConversationsViewResponse, nil
}

type conversationHistoryForm struct {
	BaseRequest
	Channel                      string         `json:"channel,omitempty"`
	Limit                        int            `json:"limit,omitempty"`
	IgnoreReplies                bool           `json:"ignore_replies,omitempty"`
	IncludePinCount              bool           `json:"include_pin_count,omitempty"`
	Inclusive                    bool           `json:"inclusive,omitempty"`
	NoUserProfile                bool           `json:"no_user_profile,omitempty"`
	IncludeStories               bool           `json:"include_stories,omitempty"`
	IncludeFreeTeamExtraMessages bool           `json:"include_free_team_extra_messages,omitempty"`
	IncludeDateJoined            bool           `json:"include_date_joined,omitempty"`
	Oldest                       string         `json:"oldest,omitempty"` //TODO
	Latest                       string         `json:"latest,omitempty"`
	Cursor                       string         `json:"cursor,omitempty"`
	CachedLatestUpdates          map[string]any `json:"cached_latest_updates,omitempty"`
	WebClientFields
}

type ConversationsHistoryResponse struct {
	baseResponse
	LatestUpdates       map[fasttime.Time]fasttime.Time `json:"latest_updates"`
	UnchangedMessages   []fasttime.Time                 `json:"unchanged_messages"`
	Latest              fasttime.Time                   `json:"latest"`
	Messages            []slack.Message                 `json:"messages"`
	HasMore             bool                            `json:"has_more"`
	PinCount            int                             `json:"pin_count"`
	ChannelActionsTS    any                             `json:"channel_actions_ts,omitempty"` // TODO: type?
	ChannelActionsCount int                             `json:"channel_actions_count,omitempty"`
}

type ConversationsHistoryParams struct {
	ChannelID     string
	Oldest        string
	Latest        string
	Cursor        string
	Limit         int
	IgnoreReplies bool
	Inclusive     bool
}

// ConversationsHistory retrieves the history of a conversation.
func (cl *Client) ConversationsHistory(ctx context.Context, params ConversationsHistoryParams) (*ConversationsHistoryResponse, error) {
	ctx, task := trace.NewTask(ctx, "ConversationsHistory")
	defer task.End()

	if params.Limit == 0 {
		params.Limit = safeLimit
	}
	form := conversationHistoryForm{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		Channel:                      params.ChannelID,
		Limit:                        params.Limit,
		IgnoreReplies:                params.IgnoreReplies,
		IncludePinCount:              true,
		Inclusive:                    params.Inclusive,
		NoUserProfile:                true,
		IncludeStories:               true,
		IncludeFreeTeamExtraMessages: true,
		IncludeDateJoined:            true,
		Oldest:                       params.Oldest,
		Latest:                       params.Latest,
		CachedLatestUpdates:          map[string]any{},
		WebClientFields:              webclientReason("message-pane/requestHistory"),
	}

	resp, err := cl.PostForm(ctx, "conversations.history", values(form, true))
	if err != nil {
		return nil, err
	}

	var r ConversationsHistoryResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, err
	}
	return &r, nil
}

type conversationsRepliesForm struct {
	BaseRequest
	Channel             string            `json:"channel,omitempty"`
	TS                  string            `json:"ts,omitempty"`
	Inclusive           bool              `json:"inclusive,omitempty"`
	Limit               int               `json:"limit,omitempty"`
	Oldest              string            `json:"oldest,omitempty"`
	Latest              string            `json:"latest,omitempty"`
	CachedLatestUpdates map[string]string `json:"cached_latest_updates,omitempty"`
	WebClientFields
}

type ConversationsRepliesParams struct {
	Channel   string
	TS        string
	Limit     int
	Oldest    string
	Latest    string
	Inclusive bool
}

type ConversationsRepliesResponse struct {
	baseResponse
	LatestUpdates     map[fasttime.Time]fasttime.Time `json:"latest_updates,omitempty"`
	UnchangedMessages []fasttime.Time                 `json:"unchanged_messages,omitempty"`
	Messages          []slack.Message                 `json:"messages,omitempty"`
	HasMore           bool                            `json:"has_more,omitempty"`
}

func (cl *Client) ConversationsReplies(ctx context.Context, params ConversationsRepliesParams) (*ConversationsRepliesResponse, error) {
	ctx, task := trace.NewTask(ctx, "ConversationsReplies")
	defer task.End()

	if params.Limit == 0 {
		params.Limit = safeLimit
	}
	form := conversationsRepliesForm{
		BaseRequest: BaseRequest{
			Token: cl.token,
		},
		Channel:             params.Channel,
		TS:                  params.TS,
		Inclusive:           params.Inclusive,
		Limit:               params.Limit,
		Oldest:              params.Oldest,
		Latest:              params.Latest,
		CachedLatestUpdates: map[string]string{},
		WebClientFields:     webclientReason("history-api/fetchReplies"),
	}
	resp, err := cl.PostForm(ctx, "conversations.replies", values(form, true))
	if err != nil {
		return nil, fmt.Errorf("conversations.replies: %w", err)
	}
	var r ConversationsRepliesResponse
	if err := cl.ParseResponse(&r, resp); err != nil {
		return nil, fmt.Errorf("conversations.replies: %w", err)
	}
	return &r, nil
}
