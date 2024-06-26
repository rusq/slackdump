package cache

import (
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/types"
)

// loadUsers tries to load the users from the file. If the file does not exist
// or is older than maxAge, it returns an error.
func loadChannels(cacheDir, filename string, suffix string, maxAge time.Duration) ([]slack.Channel, error) {
	uu, err := load[slack.Channel](cacheDir, filename, suffix, maxAge)
	if err != nil {
		return nil, err
	}
	return types.Channels(uu), nil
}

// saveUsers saves the users to a file, naming the file based on the filename
// and the suffix. The file will be saved in the cache directory.
func saveChannels(cacheDir, filename string, suffix string, cc []slack.Channel) error {
	return save(cacheDir, filename, suffix, cc)
}
