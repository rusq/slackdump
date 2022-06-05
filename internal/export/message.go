package export

import (
	"sort"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/types"
)

// ExportMessage is the slack.Message with additional fields usually found in
// slack exports.
type ExportMessage struct {
	slack.Msg

	// additional fields not defined by the slack library, but present
	// in slack exports
	UserTeam        string             `json:"user_team,omitempty"`
	SourceTeam      string             `json:"source_team,omitempty"`
	UserProfile     *ExportUserProfile `json:"user_profile,omitempty"`
	ReplyUsersCount int                `json:"reply_users_count,omitempty"`
	ReplyUsers      []string           `json:"reply_users,omitempty"`
}

type ExportUserProfile struct {
	AvatarHash        string `json:"avatar_hash,omitempty"`
	Image72           string `json:"image_72,omitempty"`
	FirstName         string `json:"first_name,omitempty"`
	RealName          string `json:"real_name,omitempty"`
	DisplayName       string `json:"display_name,omitempty"`
	Team              string `json:"team,omitempty"`
	Name              string `json:"name,omitempty"`
	IsRestricted      bool   `json:"is_restricted,omitempty"`
	IsUltraRestricted bool   `json:"is_ultra_restricted,omitempty"`
}

func (em ExportMessage) Time() time.Time {
	ts, _ := structures.ParseSlackTS(em.Timestamp)
	return ts
}

// newExportMessage creates an export message from a slack message and populates
// some additional fields.  Slack messages produced by export are much more
// saturated with information, i.e. contain user profiles and thread stats.
func newExportMessage(msg *types.Message, users structures.UserIndex) *ExportMessage {
	expMsg := ExportMessage{Msg: msg.Msg}

	expMsg.UserTeam = msg.Team
	expMsg.SourceTeam = msg.Team

	if user, ok := users[msg.User]; ok && !user.IsBot {
		expMsg.UserProfile = &ExportUserProfile{
			AvatarHash:        user.Profile.AvatarHash, // is currently not populated.
			Image72:           user.Profile.Image72,
			FirstName:         user.Profile.FirstName,
			RealName:          user.Profile.RealName,
			DisplayName:       user.Profile.DisplayName,
			Team:              user.Profile.Team,
			Name:              user.Name,
			IsRestricted:      user.IsRestricted,
			IsUltraRestricted: user.IsUltraRestricted,
		}
	}

	if !msg.IsThreadParent() {
		return &expMsg
	}

	// threaded message branch

	for _, replyMsg := range msg.ThreadReplies {
		expMsg.Msg.Replies = append(expMsg.Msg.Replies, slack.Reply{User: replyMsg.User, Timestamp: replyMsg.Timestamp})
		expMsg.ReplyUsers = append(expMsg.ReplyUsers, replyMsg.User)
	}

	sort.Slice(expMsg.Msg.Replies, func(i, j int) bool {
		tsi, _ := structures.ParseSlackTS(expMsg.Msg.Replies[i].Timestamp)
		tsj, _ := structures.ParseSlackTS(expMsg.Msg.Replies[j].Timestamp)
		return tsi.Before(tsj)
	})
	makeUniq(&expMsg.ReplyUsers)
	expMsg.ReplyUsersCount = len(expMsg.ReplyUsers)

	return &expMsg
}

// makeUniq scans the slice ss, removes all duplicates and sorts it.  Case
// sensitive.
func makeUniq(ss *[]string) {
	var uniq = make(map[string]bool, len(*ss))

	for _, s := range *ss {
		if uniq[s] {
			continue
		}
		uniq[s] = true
	}
	keys := make([]string, 0, len(uniq))
	for k := range uniq {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	*ss = keys
}
