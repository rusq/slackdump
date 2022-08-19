package edge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type UsersListRequest struct {
	BaseRequest
	Channels     []string `json:"channels"`
	PresentFirst bool     `json:"present_first"`
	Filter       string   `json:"filter"`
	Count        int64    `json:"count"`
}

type UsersListResponse struct {
	Results []User `json:"results"`
	BaseResponse
}

type User struct {
	ID                     string  `json:"id"`
	TeamID                 string  `json:"team_id"`
	Name                   string  `json:"name"`
	Deleted                bool    `json:"deleted"`
	Color                  string  `json:"color"`
	RealName               string  `json:"real_name"`
	Tz                     string  `json:"tz"`
	TzLabel                string  `json:"tz_label"`
	TzOffset               int64   `json:"tz_offset"`
	Profile                Profile `json:"profile"`
	IsAdmin                bool    `json:"is_admin"`
	IsOwner                bool    `json:"is_owner"`
	IsPrimaryOwner         bool    `json:"is_primary_owner"`
	IsRestricted           bool    `json:"is_restricted"`
	IsUltraRestricted      bool    `json:"is_ultra_restricted"`
	IsBot                  bool    `json:"is_bot"`
	IsAppUser              bool    `json:"is_app_user"`
	Updated                int64   `json:"updated"`
	IsEmailConfirmed       bool    `json:"is_email_confirmed"`
	WhoCanShareContactCard string  `json:"who_can_share_contact_card"`
}

type Profile struct {
	Title                  string  `json:"title"`
	Phone                  string  `json:"phone"`
	Skype                  string  `json:"skype"`
	RealName               string  `json:"real_name"`
	RealNameNormalized     string  `json:"real_name_normalized"`
	DisplayName            string  `json:"display_name"`
	DisplayNameNormalized  string  `json:"display_name_normalized"`
	Fields                 any     `json:"fields"`
	StatusText             string  `json:"status_text"`
	StatusEmoji            string  `json:"status_emoji"`
	StatusEmojiDisplayInfo []any   `json:"status_emoji_display_info"`
	StatusExpiration       int64   `json:"status_expiration"`
	AvatarHash             string  `json:"avatar_hash"`
	GuestInvitedBy         string  `json:"guest_invited_by"`
	ImageOriginal          *string `json:"image_original,omitempty"`
	IsCustomImage          *bool   `json:"is_custom_image,omitempty"`
	Email                  string  `json:"email"`
	FirstName              *string `json:"first_name,omitempty"`
	LastName               *string `json:"last_name,omitempty"`
	StatusTextCanonical    string  `json:"status_text_canonical"`
	Team                   string  `json:"team"`
}

type UsersInfoRequest struct {
	BaseRequest
	CheckInteraction        bool             `json:"check_interaction"`
	IncludeProfileOnlyUsers bool             `json:"include_profile_only_users"`
	UpdatedIDS              map[string]int64 `json:"updated_ids"`
}

type UserInfoResponse struct {
	Results     []UserInfo      `json:"results"`
	FailedIDS   []string        `json:"failed_ids"`
	PendingIDS  []string        `json:"pending_ids"`
	CanInteract map[string]bool `json:"can_interact"`
	BaseResponse
}

type UserInfo struct {
	ID                     string  `json:"id"`
	TeamID                 string  `json:"team_id"`
	Name                   string  `json:"name"`
	Color                  string  `json:"color"`
	IsBot                  bool    `json:"is_bot"`
	IsAppUser              bool    `json:"is_app_user"`
	Deleted                bool    `json:"deleted"`
	Profile                Profile `json:"profile"`
	IsStranger             bool    `json:"is_stranger"`
	Updated                int64   `json:"updated"`
	WhoCanShareContactCard string  `json:"who_can_share_contact_card"`
}

type UserMembershipRequest struct {
	BaseRequest
	Channel string   `json:"channel"`
	Users   []string `json:"users"`
	AsAdmin bool     `json:"as_admin"`
}

type UserMembershipResponse struct {
	Channel    string   `json:"channel"`
	NonMembers []string `json:"non_members"`
	BaseResponse
}

var ErrNotOK = errors.New("server returned NOT OK")

// GetUsers returns users from the slack edge api for the channel.  User IDs
// should be provided by the caller.  If ids is empty, does nothing.
//
// This tries to replicate the logic of the Slack client, when it fetches
// the channel users while being logged in as a guest user.
func (cl *Client) GetUsers(ctx context.Context, ids []string) ([]UserInfo, error) {
	if len(ids) == 0 {
		return []UserInfo{}, nil
	}
	var updatedIds = make(map[string]int64, len(ids))
	for _, id := range ids {
		updatedIds[id] = 0
	}

	var users []UserInfo
	for {
		uiresp, err := cl.UsersInfo(ctx, &UsersInfoRequest{
			CheckInteraction:        true,
			IncludeProfileOnlyUsers: true,
			UpdatedIDS:              updatedIds,
		})
		if err != nil {
			return nil, err
		}
		if !uiresp.Ok {
			return nil, ErrNotOK
		}
		if len(uiresp.Results) > 0 {
			users = append(users, uiresp.Results...)
		}
		if len(uiresp.PendingIDS) == 0 {
			break
		}
		for _, ui := range uiresp.Results {
			updatedIds[ui.ID] = ui.Updated
		}
	}
	return users, nil
}

// UsersInfo calls a users.info endpoint.  This endpoint does not return results
// straight away.  It may return "pending ids", and when it does, it should be
// called again to get the actual user info (see [Client.GetUsers]).
func (cl *Client) UsersInfo(ctx context.Context, req *UsersInfoRequest) (*UserInfoResponse, error) {
	var ui UserInfoResponse
	if err := callAPI(ctx, cl, "users/info", req, &ui); err != nil {
		return nil, err
	}
	return &ui, nil
}

// ChannelMembership calls channels.membership endpoint.
func (cl *Client) ChannelMembership(ctx context.Context, req *UserMembershipRequest) (*UserMembershipResponse, error) {
	var um UserMembershipResponse
	if err := callAPI(ctx, cl, "channels/membership", req, &um); err != nil {
		return nil, err
	}
	return &um, nil
}

func callAPI(ctx context.Context, cl *Client, endpoint string, req PostRequest, resp any) error {
	r, err := cl.Post(ctx, endpoint, req)
	if err != nil {
		return err
	}
	if err != nil && err != io.EOF {
		return err
	}
	if r.StatusCode < http.StatusOK || http.StatusMultipleChoices <= r.StatusCode {
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		return fmt.Errorf("error:  status code: %s, body: %s", r.Status, string(body))
	}
	return cl.ParseResponse(resp, r)
}
