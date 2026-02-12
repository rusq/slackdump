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
	"testing"
	"time"

	"github.com/rusq/slack"
)

var testThread = []slack.Message{
	{Msg: slack.Msg{Timestamp: "1700000000.000000", ThreadTimestamp: "1700000000.000000", User: "UBOB"}},
	{Msg: slack.Msg{Timestamp: "1710000000.000000", ThreadTimestamp: "1700000000.000000", User: "UALICE"}},
	{Msg: slack.Msg{Timestamp: "1720000000.000000", ThreadTimestamp: "1700000000.000000", User: "UBOB"}},
	{Msg: slack.Msg{Timestamp: "1730000000.000000", ThreadTimestamp: "1700000000.000000", User: "UDAVE"}},
	{Msg: slack.Msg{Timestamp: "1740000000.000000", ThreadTimestamp: "1700000000.000000", User: "UCHARLIE"}},
	{Msg: slack.Msg{Timestamp: "1750000000.000000", ThreadTimestamp: "1700000000.000000", User: "UBOB"}},
}

func TestExportMessage_PopulateReplyFields(t *testing.T) {
	type fields struct {
		Msg             *slack.Msg
		UserTeam        string
		SourceTeam      string
		UserProfile     *ExportUserProfile
		ReplyUsersCount int
		slackdumpTime   time.Time
	}
	type args struct {
		thread []slack.Message
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ExportMessage
	}{
		{
			name:   "Zero thread length",
			fields: fields{},
			args: args{
				thread: []slack.Message{},
			},
			want: &ExportMessage{},
		},
		{
			name: "Is not a lead message",
			fields: fields{
				Msg: &slack.Msg{
					Timestamp: "123",
				},
			},
			args: args{thread: testThread},
			want: &ExportMessage{},
		},
		{
			name: "Is a lead message",
			fields: fields{
				Msg: &testThread[0].Msg,
			},
			args: args{thread: testThread},
			want: &ExportMessage{
				Msg: &slack.Msg{
					Timestamp:       testThread[0].Timestamp,
					ThreadTimestamp: testThread[0].ThreadTimestamp,
					ReplyUsers:      []string{"UALICE", "UBOB", "UCHARLIE", "UDAVE"},
					Replies: []slack.Reply{
						{User: "UBOB", Timestamp: "1700000000.000000"},
						{User: "UALICE", Timestamp: "1710000000.000000"},
						{User: "UBOB", Timestamp: "1720000000.000000"},
						{User: "UDAVE", Timestamp: "1730000000.000000"},
						{User: "UCHARLIE", Timestamp: "1740000000.000000"},
						{User: "UBOB", Timestamp: "1750000000.000000"},
					},
				},
				ReplyUsersCount: 4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &ExportMessage{
				Msg:             tt.fields.Msg,
				UserTeam:        tt.fields.UserTeam,
				SourceTeam:      tt.fields.SourceTeam,
				UserProfile:     tt.fields.UserProfile,
				ReplyUsersCount: tt.fields.ReplyUsersCount,
				slackdumpTime:   tt.fields.slackdumpTime,
			}
			em.PopulateReplyFields(tt.args.thread)
		})
	}
}

func BenchmarkExportMessage_PopulateReplyFields(b *testing.B) {
	em := &ExportMessage{
		Msg: &testThread[0].Msg,
	}
	for i := 0; i < b.N; i++ {
		em.PopulateReplyFields(testThread)
	}
}
