package types

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/structures"
)

// Channels keeps slice of channels.
type Channels []slack.Channel

// ToText outputs Channels to w in text format.
func (cs Channels) ToText(w io.Writer, ui structures.UserIndex) (err error) {
	const strFormat = "%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintf(writer, strFormat, "ID", "Arch", "Saved", "What")
	for i, ch := range cs {
		who := ui.ChannelName(&ch)
		archived := "-"
		if cs[i].IsArchived || ui.IsDeleted(ch.User) {
			archived = "arch"
		}
		saved := "-"
		if _, err := os.Stat(ch.ID + ".json"); err == nil {
			saved = "saved"
		}

		fmt.Fprintf(writer, strFormat, ch.ID, archived, saved, who)
	}
	return nil
}

// UserIDs returns a slice of user IDs.
func (c Channels) UserIDs() []string {
	var seen = make(map[string]bool, len(c))
	for _, m := range c {
		if m.User == "" {
			if seen[m.Creator] {
				continue
			}
			seen[m.Creator] = true
		}
		if seen[m.User] {
			continue
		}
		seen[m.User] = true
	}
	return toslice(seen)
}
