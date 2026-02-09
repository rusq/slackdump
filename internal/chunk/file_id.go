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

package chunk

import (
	"strings"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// FileID is the ID of the file within the Directory (it's basically the file
// name without an extension).
type FileID string

// chanThreadSep is the separator between channel name and a thread name in
// the file ID.
const chanThreadSep = "-"

// ToFileID returns the file ID for the given channel and thread timestamp.
// If includeThread is true and threadTS is not empty, the thread timestamp
// will be appended to the channel ID.  Otherwise, only the channel ID will be
// returned.
func ToFileID(channelID, threadTS string, includeThread bool) FileID {
	if includeThread && threadTS != "" {
		return FileID(channelID + chanThreadSep + threadTS)
	}
	return FileID(channelID)
}

// LinkToFileID converts the SlackLink to file ID.  If includeThread is true
// and the thread timestamp is not empty, the thread timestamp will be
// appended to the channel ID.  Otherwise, only the channel ID will be
// returned.
func LinkToFileID(sl structures.SlackLink, includeThread bool) FileID {
	return ToFileID(sl.Channel, sl.ThreadTS, includeThread)
}

// Split splits the file ID into channel ID and thread timestamp.  If the file
// ID doesn't contain the thread timestamp, the thread timestamp will be
// empty.
func (id FileID) Split() (channelID, threadTS string) {
	channelID, threadTS, _ = strings.Cut(string(id), chanThreadSep)
	return
}

// SlackLink returns the SlackLink for the file ID.  If the file ID doesn't
// contain the thread timestamp, the thread timestamp will be empty.
func (id FileID) SlackLink() structures.SlackLink {
	channelID, threadTS := id.Split()
	return structures.SlackLink{Channel: channelID, ThreadTS: threadTS}
}

func (id FileID) String() string {
	return string(id)
}
