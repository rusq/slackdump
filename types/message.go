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

import (
	"sort"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
)

// Message is the internal representation of message with thread.
type Message struct {
	slack.Message
	ThreadReplies []Message `json:"slackdump_thread_replies,omitempty"`
}

func (m Message) Datetime() (time.Time, error) {
	return structures.ParseSlackTS(m.Timestamp)
}

// IsBotMessage returns true if the message is from a bot.
func (m Message) IsBotMessage() bool {
	return m.BotID != ""
}

func (m Message) IsThread() bool {
	return m.ThreadTimestamp != ""
}

// IsThreadParent will return true if the message is the parent message of a
// conversation (has more than 0 replies)
func (m Message) IsThreadParent() bool {
	return m.IsThread() && m.ReplyCount != 0
}

// IsThreadChild will return true if the message is the child message of a
// conversation.
func (m Message) IsThreadChild() bool {
	return m.IsThread() && m.ReplyCount == 0
}

func SortMessages(msgs []Message) {
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Timestamp < msgs[j].Timestamp
	})
}

// ConvertMsgs converts a slice of slack.Message to []types.Message.
func ConvertMsgs(sm []slack.Message) []Message {
	msgs := make([]Message, len(sm))
	for i := range sm {
		msgs[i].Message = sm[i]
	}
	return msgs
}
