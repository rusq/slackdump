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
	*slack.Msg

	// additional fields not defined by the slack library, but present
	// in slack exports
	UserTeam        string             `json:"user_team"`
	SourceTeam      string             `json:"source_team"`
	UserProfile     *ExportUserProfile `json:"user_profile"`
	ReplyUsersCount int                `json:"reply_users_count"`
	slackdumpTime   time.Time          `json:"-"`
}

type ExportUserProfile struct {
	AvatarHash        string `json:"avatar_hash"`
	Image72           string `json:"image_72"`
	FirstName         string `json:"first_name"`
	RealName          string `json:"real_name"`
	DisplayName       string `json:"display_name"`
	Team              string `json:"team"`
	Name              string `json:"name"`
	IsRestricted      bool   `json:"is_restricted"`
	IsUltraRestricted bool   `json:"is_ultra_restricted"`
}

func (em ExportMessage) Time() time.Time {
	if em.slackdumpTime.IsZero() {
		ts, _ := structures.ParseSlackTS(em.Timestamp)
		return ts
	}
	return em.slackdumpTime
}

// newExportMessage creates an export message from a slack message and populates
// some additional fields.  Slack messages produced by export are much more
// saturated with information, i.e. contain user profiles and thread stats.
func newExportMessage(msg *types.Message, users structures.UserIndex) *ExportMessage {
	if msg == nil {
		panic("internal error: msg is nil")
	}
	expMsg := ExportMessage{Msg: &msg.Msg}

	expMsg.UserTeam = msg.Team
	expMsg.SourceTeam = msg.Team
	expMsg.slackdumpTime, _ = msg.Datetime()

	if user, ok := users[msg.User]; ok && !user.IsBot {
		expMsg.UserProfile = &ExportUserProfile{
			AvatarHash:        "",
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

	expMsg.Replies = make([]slack.Reply, 0, len(msg.ThreadReplies))
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
