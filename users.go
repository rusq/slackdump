package slackdump

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/slack-go/slack"
)

// Users keeps slice of users
type Users struct {
	Users []slack.User
	SD    *SlackDumper
}

// GetUsers retrieve all users and refresh structure User information
func (sd *SlackDumper) GetUsers() (*Users, error) {
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
	sd.Users.Users = users
	sd.UserForID = sd.Users.MakeUserIDIndex()

	return &Users{Users: users, SD: sd}, err
}

// ToText outputs Users us to io.Writer w in Text format
func (us Users) ToText(w io.Writer) (err error) {
	//const strFormat = "%-*s (id: %8s) %3s %8s\n"
	const strFormat = "%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	var names = make([]string, 0, len(us.Users))
	var usermap = make(map[string]*slack.User, len(us.Users))
	for i := range us.Users {
		names = append(names, us.Users[i].Name)
		usermap[us.Users[i].Name] = &us.Users[i]
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

// MakeUserIDIndex returns the userID map to relevant *slack.User
func (us *Users) MakeUserIDIndex() map[string]*slack.User {
	var usermap = make(map[string]*slack.User, len(us.Users))
	// that's for readability
	var userlist = &us.Users

	for i := range *userlist {
		usermap[(*userlist)[i].ID] = &(*userlist)[i]
	}

	return usermap
}

// Len returns length of underlying []slack.Users slice
func (us *Users) Len() int {
	return len(us.Users)
}
