package structures

import (
	"errors"
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

const (
	mpimNameSep = "--"
	mpimPrefix  = "mpdm-"
)

// FixMpIMmembers verifies Channel.Members to contain all channel members, if
// not, it attempts to populate it by parsing out the usernames from
// name_normalized, and populating Channel.Members with IDs of the users
// discovered.  usernameIDs is a mapping of username -> user_id.
func FixMpIMmembers(ch *slack.Channel, users []slack.User) (*slack.Channel, error) {
	if !isMpIM(ch) {
		return nil, fmt.Errorf("not a MpIM channel: %s (IsMpIM=%v)", ch.ID, ch.IsMpIM)
	}
	if len(ch.Members) == mpimMemberCount(ch.NameNormalized) {
		return ch, nil
	}
	members, err := parseMpIMmembers(ch.NameNormalized, usernameIDs(users))
	if err != nil {
		return nil, err
	}
	ch.Members = members
	return ch, nil
}

func mpimMemberCount(nameNormalized string) int {
	return strings.Count(nameNormalized, mpimNameSep) + 1
}

func isMpIM(ch *slack.Channel) bool {
	return ch.IsMpIM && strings.HasPrefix(ch.NameNormalized, mpimPrefix)
}

func parseMpIMmembers(nn string, usernameIDs map[string]string) ([]string, error) {
	if mpimMemberCount(nn) == 0 {
		return nil, errors.New("no members in mpim")
	}
	if len(usernameIDs) == 0 {
		return nil, errors.New("no user mapping")
	}
	nn = strings.TrimRight(strings.TrimLeft(nn, mpimPrefix), "-1")
	names := strings.Split(nn, mpimNameSep)
	var members = make([]string, len(names))
	for i := range names {
		var ok bool
		members[i], ok = usernameIDs[names[i]]
		if !ok {
			// TODO: log?
			continue
		}
	}
	return members, nil
}

// UsernameIDs returns a mapping of user.name->user.id.
func usernameIDs(us []slack.User) map[string]string {
	var ui = make(map[string]string)
	for _, u := range us {
		ui[u.Name] = u.ID
	}
	return ui
}
