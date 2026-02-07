// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
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
