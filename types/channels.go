package types

import (
	"github.com/rusq/slack"
)

// Channels keeps slice of channels.
type Channels []slack.Channel

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
