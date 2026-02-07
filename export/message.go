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
package export

import (
	"slices"
	"sort"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
)

// ExportMessage is the slack.Message with additional fields usually found in
// slack exports.
type ExportMessage struct {
	*slack.Msg

	// additional fields not defined by the slack library, but present
	// in slack exports
	UserTeam        string             `json:"user_team,omitempty"`
	SourceTeam      string             `json:"source_team,omitempty"`
	UserProfile     *ExportUserProfile `json:"user_profile,omitempty"`
	ReplyUsersCount int                `json:"reply_users_count,omitempty"`
	slackdumpTime   time.Time          `json:"-"` // to speedup sorting
}

type ExportUserProfile struct {
	AvatarHash        string `json:"avatar_hash"`
	Image72           string `json:"image_72"`
	FirstName         string `json:"first_name"`
	RealName          string `json:"real_name"`
	DisplayName       string `json:"display_name"`
	Team              string `json:"team"`
	Name              string `json:"name"`
	IsRestricted      bool   `json:"is_restricted"`
	IsUltraRestricted bool   `json:"is_ultra_restricted"`
}

func (em *ExportMessage) Time() time.Time {
	if em.slackdumpTime.IsZero() {
		ts, _ := structures.ParseSlackTS(em.Timestamp)
		return ts
	}
	return em.slackdumpTime
}

// SlackMessage returns the slack.Message.
func (em *ExportMessage) SlackMessage() *slack.Message {
	return &slack.Message{
		Msg: *em.Msg,
	}
}

// reply is the special type to sort the replies by timestamp faster.
type reply struct {
	slack.Reply
	ts int64
}

func (em *ExportMessage) PopulateReplyFields(thread []slack.Message) {
	if len(thread) == 0 || !structures.IsThreadStart(em.SlackMessage()) {
		// reply fields are only populated on the lead message of a thread.
		return
	}
	if thread[0].ThreadTimestamp == thread[0].Timestamp {
		thread = thread[1:] // remove lead message from the start
	} else if thread[len(thread)-1].ThreadTimestamp == thread[0].Timestamp {
		thread = thread[:len(thread)-1] // remove lead message from the end
	}

	replyUsers := make(map[string]struct{}, len(thread))
	replies := make([]reply, len(thread))
	for i := range thread {
		replies[i].User = thread[i].User
		replies[i].Timestamp = thread[i].Timestamp
		replies[i].ts, _ = fasttime.TS2int(thread[i].Timestamp)
		if _, ok := replyUsers[thread[i].User]; !ok {
			replyUsers[thread[i].User] = struct{}{}
		}
	}
	sort.Slice(replies, func(i, j int) bool {
		return replies[i].ts < replies[j].ts
	})
	em.Replies = make([]slack.Reply, len(replies))
	for i := range replies {
		em.Replies[i] = replies[i].Reply
	}

	em.ReplyUsersCount = len(replyUsers)
	em.ReplyUsers = make([]string, 0, em.ReplyUsersCount)
	for k := range replyUsers {
		em.ReplyUsers = append(em.ReplyUsers, k)
	}
	slices.Sort(em.ReplyUsers)
}
