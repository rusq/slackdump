package types

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/structures"
)

// Users is a slice of users.
type Users []slack.User

// ToText outputs Users us to io.Writer w in Text format
func (us Users) ToText(w io.Writer, _ structures.UserIndex) error {
	const strFormat = "%s\t%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()

	// header
	if _, err := fmt.Fprintf(writer, strFormat, "Name", "ID", "Bot?", "Deleted?", "Restricted?"); err != nil {
		return fmt.Errorf("writer error: %w", err)
	}
	if _, err := fmt.Fprintf(writer, strFormat, "", "", "", "", ""); err != nil {
		return fmt.Errorf("writer error: %w", err)
	}

	var (
		names   = make([]string, 0, len(us))
		usermap = make(structures.UserIndex, len(us))
	)
	for i := range us {
		names = append(names, us[i].Name)
		usermap[us[i].Name] = &us[i]
	}
	sort.Strings(names)

	// data
	for _, name := range names {
		var (
			deleted    string
			bot        string
			restricted string
		)
		if usermap[name].Deleted {
			deleted = "deleted"
		}
		if usermap[name].IsBot {
			bot = "bot"
		}
		if usermap[name].IsRestricted {
			restricted = "restricted"
		}

		_, err := fmt.Fprintf(writer, strFormat,
			name, usermap[name].ID, bot, deleted, restricted,
		)
		if err != nil {
			return fmt.Errorf("writer error: %w", err)
		}
	}
	return nil
}

// IndexByID returns the userID map to relevant *slack.User
func (us Users) IndexByID() structures.UserIndex {
	return structures.NewUserIndex(us)
}
