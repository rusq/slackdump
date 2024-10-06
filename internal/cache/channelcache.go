package cache

import (
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/types"
)

// loadChannels tries to load channels from the file. If the file does not exist
// or is older than maxAge, it returns an error.
func loadChannels(dirname, filename string, suffix string, maxAge time.Duration) ([]slack.Channel, error) {
	uu, err := load[slack.Channel](dirname, filename, suffix, maxAge)
	if err != nil {
		return nil, err
	}
	return types.Channels(uu), nil
}

// saveChannels saves channels to a file, naming the file based on the
// filename and the suffix. The file will be saved in the dirname.
func saveChannels(dirname, filename string, suffix string, cc []slack.Channel) error {
	return save(dirname, filename, suffix, cc)
}
