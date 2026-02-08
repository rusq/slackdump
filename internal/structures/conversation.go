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

// IsThreadStart check if the message is a lead message of a thread and the
// thread is a non-empty thread.
func IsThreadStart(m *slack.Message) bool {
	return m.ThreadTimestamp != "" && m.Timestamp == m.ThreadTimestamp && !IsEmptyThread(m)
}

// IsEmptyThread checks if the message is a thread with no replies.
func IsEmptyThread(m *slack.Message) bool {
	return m.LatestReply == LatestReplyNoReplies
}

// IsThreadMessage checks if the message is a thread message (not lead).
func IsThreadMessage(m *slack.Msg) bool {
	return m.ThreadTimestamp != "" && m.ThreadTimestamp != m.Timestamp
}

const (
	CMPIM    = "mpim"            // Group IM
	CIM      = "im"              // IM
	CPublic  = "public_channel"  // Public Channel
	CPrivate = "private_channel" // Private Channel
)

func ChannelType(ch slack.Channel) string {
	switch {
	case ch.IsIM:
		return CIM
	case ch.IsMpIM:
		return CMPIM
	case ch.IsPrivate:
		return CPrivate
	default:
		return CPublic
	}
}

func ChannelFromID(id string) *slack.Channel {
	return &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: id}}} // arrgh
}
