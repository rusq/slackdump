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

package structures

import "github.com/rusq/slack"

// CanvasFileID returns the fileID and true if the channel has a canvas
// associated with it.
func CanvasFileID(channel *slack.Channel) (fileID string, ok bool) {
	if channel != nil &&
		channel.Properties != nil &&
		!channel.Properties.Canvas.IsEmpty &&
		channel.Properties.Canvas.FileId != "" {
		return channel.Properties.Canvas.FileId, true
	}
	return "", false
}

// CanvasChannelID returns the channelID and true if the fileID is a valid
// canvas file ID.
func CanvasChannelID(fileID string) (channelID string, ok bool) {
	if len(fileID) < 2 || fileID[0] != 'F' {
		return "", false
	}
	return "C" + fileID[1:], true
}
