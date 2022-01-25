package slackdump

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/slack-go/slack"
)

// Users keeps slice of users
type Users []slack.User

// GetUsers retrieve all users and refresh structure User information
func (sd *SlackDumper) GetUsers() (Users, error) {
	var err error
	users, err := sd.client.GetUsers()
	if err != nil {
		return nil, err
	}
	// BUG: as of 201902 there's a bug in slack module, the invalid_auth error
	// is not propagated properly, so we'll check for number of users.  There
	// should be at least one (slackbot).
	if len(users) == 0 {
		err = fmt.Errorf("couldn't fetch users")
	}
	// recalculating userForID
	sd.Users = users
	sd.UserForID = sd.Users.IndexByID()

	return users, err
}

// ToText outputs Users us to io.Writer w in Text format
func (us Users) ToText(w io.Writer) (err error) {
	//const strFormat = "%-*s (id: %8s) %3s %8s\n"
	const strFormat = "%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	var names = make([]string, 0, len(us))
	var usermap = make(map[string]*slack.User, len(us))
	for i := range us {
		names = append(names, us[i].Name)
		usermap[us[i].Name] = &us[i]
	}
	sort.Strings(names)

	// maxNameLen := maxStringLength(names)
	// header
	fmt.Fprintf(writer, strFormat, "Name", "ID", "Bot?", "Deleted?")
	fmt.Fprintf(writer, strFormat, "", "", "", "")

	for _, name := range names {
		var deleted, bot string
		if usermap[name].Deleted {
			deleted = "deleted"
		}
		if usermap[name].IsBot {
			bot = "bot"
		}

		fmt.Fprintf(writer, strFormat,
			name, usermap[name].ID, bot, deleted)
	}
	return
}

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() map[string]*slack.User {
	var usermap = make(map[string]*slack.User, len(us))

	for i := range us {
		usermap[(us)[i].ID] = &us[i]
	}

	return usermap
}

// Len returns the user count
func (us Users) Len() int {
	return len(us)
}
