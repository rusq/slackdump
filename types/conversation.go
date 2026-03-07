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

package types

// Conversation keeps the slice of messages.
type Conversation struct {
	// ID is the channel ID.
	ID string `json:"channel_id"`
	// ThreadTS is a thread timestamp.  If it's not empty, it means that it's a
	// dump of a thread, not a channel.
	ThreadTS string `json:"thread_ts,omitempty"`
	// Name is the channel name.
	Name string `json:"name"`
	// Messages is a slice of messages.
	Messages []Message `json:"messages"`
}

func (c Conversation) String() string {
	if c.ThreadTS == "" {
		return c.ID
	}
	return c.ID + "-" + c.ThreadTS
}

// IsThread returns true if the conversation is a thread.
func (c Conversation) IsThread() bool {
	return c.ThreadTS != ""
}

// UserIDs returns a slice of user IDs.
func (c Conversation) UserIDs() []string {
	seen := make(map[string]bool, len(c.Messages))
	for _, m := range c.Messages {
		if seen[m.User] {
			continue
		}
		seen[m.User] = true
	}
	return toslice(seen)
}
