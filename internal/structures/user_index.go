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

// Username tries to resolve the username by ID. If it fails, it returns the
// user ID.  If the user is not found in index, is assumes that it is an external
// user and returns ID with "external" prefix.
func (idx UserIndex) Username(id string) string {
	return idx.userattr(id, func(user *slack.User) string {
		return user.Name
	})
}

// DisplayName tries to resolve the display name by ID. if the index is empty, it
// returns the user ID. If the user is not found in index, is assumes that it is
// an external user and returns ID with "external" prefix. If it does find the
// user and display name is unavailble, it returns the Real Name.
func (idx UserIndex) DisplayName(id string) string {
	return idx.userattr(id, func(user *slack.User) string {
		return nvl(user.Profile.DisplayName, user.RealName)
	})
}

func nvl(s string, ss ...string) string {
	if s != "" {
		return s
	}
	for _, alt := range ss {
		if alt != "" {
			return alt
		}
	}
	return "" // you got no luck at all, don't you.
}

// userattr finds the user by ID and calls a function fn with that user. If the
// user index is not initialised, it will return the ID, otherwise, if the user
// is not found in index, it will assume that the user is external, and return
// the ID with "external" prefix.
func (idx UserIndex) userattr(id string, fn func(user *slack.User) string) string {
	if idx == nil {
		// no user cache, use the IDs.
		return id
	}
	user, ok := idx[id]
	if !ok {
		return "<external>:" + id
	}
	return fn(user)
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
		return idx.DisplayName(userid)
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
