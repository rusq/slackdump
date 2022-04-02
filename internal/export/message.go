package export

import (
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

// userIndex maps the userID to a slack User.
type userIndex map[string]*slack.User

// newExportMessage populates some additional fields of a message.  Slack
// messages produced by export are much more saturated with information, i.e.
// contain user profiles and thread stats.
func newExportMessage(msg *slackdump.Message, users userIndex) *ExportMessage {
	expMsg := ExportMessage{Msg: msg.Msg}

	if len(msg.ThreadReplies) == 0 {
		return &expMsg
	}
	/*
		Parent message of a thread:
			user_team
			source_team
			user_profile
			reply_users_count
			reply_users
			replies []

		Each thread message:
			parent_user_id
	*/
	expMsg.Msg.ParentUserId = msg.User
	expMsg.UserTeam = msg.Team
	expMsg.SourceTeam = msg.Team

	for _, replyMsg := range msg.ThreadReplies {
		expMsg.Msg.Replies = append(msg.Msg.Replies, slack.Reply{User: replyMsg.User, Timestamp: replyMsg.Timestamp})
		// expMsg.UserProfile =  // create a map of users and get the user from it
		expMsg.ReplyUsers = append(expMsg.ReplyUsers, replyMsg.User) // TODO: make unique
	}
	return &expMsg
}
