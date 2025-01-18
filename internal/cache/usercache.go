package cache

import (
	"io"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/types"
)

func ReadUsers(r io.Reader) (types.Users, error) {
	uu, err := read[slack.User](r)
	if err != nil {
		return nil, err
	}
	return types.Users(uu), nil
}

// loadUsers tries to load the users from the file. If the file does not exist
// or is older than maxAge, it returns an error.
func (m *Manager) loadUsers(dirname, filename string, suffix string, maxAge time.Duration) (types.Users, error) {
	uu, err := load[slack.User](dirname, filename, suffix, maxAge, m.machineID)
	if err != nil {
		return nil, err
	}
	return types.Users(uu), nil
}

// saveUsers saves the users to a file, naming the file based on the filename
// and the suffix. The file will be saved in the cache directory.
func (m *Manager) saveUsers(dirname, filename string, suffix string, uu types.Users) error {
	return save(dirname, filename, suffix, []slack.User(uu), m.machineID)
}
