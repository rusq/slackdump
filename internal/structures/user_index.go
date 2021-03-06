package structures

import (
	"strings"

	"github.com/slack-go/slack"
)

// UserIndex is a mapping of user ID to the *slack.User.
type UserIndex map[string]*slack.User

// NewUserIndex creates a new UserIndex from slack Users slice
func NewUserIndex(us []slack.User) UserIndex {
	var usermap = make(UserIndex, len(us))

	for i := range us {
		usermap[(us)[i].ID] = &us[i]
	}

	return usermap
}

// Username tries to resolve the username by ID. If the user index is not
// initialised, it will return the ID, otherwise, if the user is not found in
// cache, it will assume that the user is external, and return the ID with
// "external" prefix.
func (idx UserIndex) Username(id string) string {
	if idx == nil {
		// no user cache, use the IDs.
		return id
	}
	user, ok := idx[id]
	if !ok {
		return "<external>:" + id
	}
	return user.Name
}

// Sender returns username for the message
func (idx UserIndex) Sender(msg *slack.Message) string {
	var userid string
	if msg.Comment != nil {
		userid = msg.Comment.User
	} else {
		userid = msg.User
	}

	if userid != "" {
		return idx.Username(userid)
	}

	return ""
}

// IsDeleted checks if the user is deleted and returns appropriate value. It
// will assume user is not deleted, if it's not present in the user index.
func (idx UserIndex) IsDeleted(id string) bool {
	thisUser, ok := idx[id]
	if !ok {
		return false
	}
	return thisUser.Deleted
}

// ChannelName return the "beautified" name of the channel.
func (idx UserIndex) ChannelName(channel *slack.Channel) (who string) {
	switch {
	case channel.IsIM:
		who = "@" + idx.Username(channel.User)
	case channel.IsMpIM:
		who = strings.Replace(channel.Purpose.Value, " messaging with", "", -1)
	case channel.IsPrivate:
		who = "🔒 " + channel.NameNormalized
	default:
		who = "#" + channel.NameNormalized
	}
	return who
}
