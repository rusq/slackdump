// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package structures

import (
	"strings"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/primitive"
)

// UserIndex is a mapping of user ID to the *slack.User.
type UserIndex map[string]*slack.User

// NewUserIndex creates a new UserIndex from slack Users slice
func NewUserIndex(us []slack.User) UserIndex {
	usermap := make(UserIndex, len(us))

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
// user and display name is unavailable, it returns the Real Name.
func (idx UserIndex) DisplayName(id string) string {
	if id == "" {
		return "Unknown User"
	}
	return idx.userattr(id, func(user *slack.User) string {
		return NVL(user.Profile.DisplayName, user.RealName, user.Name)
	})
}

func UserDisplayName(u *slack.User) string {
	return NVL(u.Name, u.RealName, u.ID)
}

func Username(u *slack.User) string {
	return NVL(u.Name, u.ID)
}

func NVL(s string, ss ...string) string {
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
func (idx UserIndex) ChannelName(ch slack.Channel) (who string) {
	t := ChannelType(ch)

	switch t {
	case CIM:
		who = "@" + idx.Username(ch.User)
	case CMPIM:
		who = strings.ReplaceAll(ch.Purpose.Value, " messaging with", "")
	case CPrivate:
		who = "🔒 " + NVL(ch.NameNormalized, ch.Name)
	default:
		who = "#" + NVL(ch.NameNormalized, ch.Name)
	}
	return who + primitive.IfTrue(ch.IsArchived, " (archived)", "")
}
