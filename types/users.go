package types

import (
	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/structures"
)

// Users is a slice of users.
type Users []slack.User

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() structures.UserIndex {
	return structures.NewUserIndex(us)
}

// UserIDs returns a slice of user IDs.
func (us Users) UserIDs() []string {
	var ids = make([]string, len(us))
	for i := range us {
		ids[i] = us[i].ID
	}
	return ids
}
