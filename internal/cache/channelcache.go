package cache

import (
	"time"

	"github.com/rusq/slack"
)

// loadChannels tries to load channels from the file. If the file does not exist
// or is older than maxAge, it returns an error.
func (m *Manager) loadChannels(dirname, filename string, suffix string, maxAge time.Duration) ([]slack.Channel, error) {
	uu, err := load[slack.Channel](dirname, filename, suffix, maxAge, m.createOpener())
	if err != nil {
		return nil, err
	}
	return uu, nil
}

// saveChannels saves channels to a file, naming the file based on the
// filename and the suffix. The file will be saved in the dirname.
func (m *Manager) saveChannels(dirname, filename string, suffix string, cc []slack.Channel) error {
	return save(dirname, filename, suffix, cc, m.createOpener())
}
