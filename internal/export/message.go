package export

import (
	"sort"
	"time"

	"github.com/rusq/slackdump"
	"github.com/slack-go/slack"
)

// ExportMessage is the slack.Message with additional fields usually found in
// slack exports.
type ExportMessage struct {
	slack.Msg
	// additional fields not defined by the library
	UserTeam        string            `json:"user_team,omitempty"`
	SourceTeam      string            `json:"source_team,omitempty"`
	UserProfile     slack.UserProfile `json:"user_profile,omitempty"`
	ReplyUsersCount int               `json:"reply_users_count,omitempty"`
	ReplyUsers      []string          `json:"reply_users,omitempty"`
}

func (em ExportMessage) Time() time.Time {
	ts, _ := slackdump.ParseSlackTS(em.Timestamp)
	return ts
}

// userIndex maps the userID to a slack User.
type userIndex map[string]*slack.User

// newExportMessage populates some additional fields of a message.  Slack
// messages produced by export are much more saturated with information, i.e.
// contain user profiles and thread stats.
func newExportMessage(msg *slackdump.Message, users userIndex) *ExportMessage {
	expMsg := ExportMessage{Msg: msg.Msg}

	expMsg.UserTeam = msg.Team
	expMsg.SourceTeam = msg.Team

	user, ok := users[msg.User]
	if ok {
		expMsg.UserProfile = user.Profile
	}

	if !msg.IsThread() {
		return &expMsg
	}

	for _, replyMsg := range msg.ThreadReplies {
		expMsg.Msg.Replies = append(msg.Msg.Replies, slack.Reply{User: replyMsg.User, Timestamp: replyMsg.Timestamp})
		expMsg.ReplyUsers = append(expMsg.ReplyUsers, replyMsg.User)
	}

	sort.Slice(expMsg.Msg.Replies, func(i, j int) bool {
		tsi, _ := slackdump.ParseSlackTS(expMsg.Msg.Replies[i].Timestamp)
		tsj, _ := slackdump.ParseSlackTS(expMsg.Msg.Replies[j].Timestamp)
		return tsi.Before(tsj)
	})
	makeUniq(&expMsg.ReplyUsers)

	return &expMsg
}

func makeUniq(ss *[]string) {
	var seen = make(map[string]bool, len(*ss))

	for _, s := range *ss {
		if seen[s] {
			continue
		}
		seen[s] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	*ss = keys
}
