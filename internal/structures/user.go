package structures

import "github.com/slack-go/slack"

type UserIndex map[string]*slack.User

// ResolveUsername tries to resolve the ResolveUsername by ID. If the internal users map is not
// initialised, it will return the ID, otherwise, if the user is not found in
// cache, it will assume that the user is external, and return the ID with
// "external" prefix.
func ResolveUsername(id string, userIdx UserIndex) string {
	if userIdx == nil {
		// no user cache, use the IDs.
		return id
	}
	user, ok := userIdx[id]
	if !ok {
		return "<external>:" + id
	}
	return user.Name
}

// SenderName returns username for the message
func SenderName(msg *slack.Message, userIdx UserIndex) string {
	var userid string
	if msg.Comment != nil {
		userid = msg.Comment.User
	} else {
		userid = msg.User
	}

	if userid != "" {
		return ResolveUsername(userid, userIdx)
	}

	return ""
}
