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

import (
	"testing"

	"github.com/rusq/slack"
)

func TestIsThreadStart(t *testing.T) {
	type args struct {
		m *slack.Message
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "thread start",
			args: args{
				m: &slack.Message{Msg: slack.Msg{Timestamp: "123", ThreadTimestamp: "123"}},
			},
			want: true,
		},
		{
			name: "thread message",
			args: args{
				m: &slack.Message{Msg: slack.Msg{Timestamp: "123", ThreadTimestamp: "456"}},
			},
			want: false,
		},
		{
			name: "no thread",
			args: args{
				m: &slack.Message{Msg: slack.Msg{Timestamp: "123", ThreadTimestamp: ""}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsThreadStart(tt.args.m); got != tt.want {
				t.Errorf("IsThreadStart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEmptyThread(t *testing.T) {
	type args struct {
		m *slack.Message
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "no replies",
			args: args{
				m: &slack.Message{Msg: slack.Msg{LatestReply: LatestReplyNoReplies}},
			},
			want: true,
		},
		{
			name: "replies",
			args: args{
				m: &slack.Message{Msg: slack.Msg{LatestReply: "123"}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEmptyThread(tt.args.m); got != tt.want {
				t.Errorf("IsEmptyThread() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChannelType(t *testing.T) {
	type args struct {
		ch slack.Channel
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "IM",
			args: args{
				ch: slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{IsIM: true}}},
			},
			want: CIM,
		},
		{
			name: "Group IM",
			args: args{
				ch: slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{IsMpIM: true}}},
			},
			want: CMPIM,
		},
		{
			name: "Private",
			args: args{
				ch: slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{IsPrivate: true}}},
			},
			want: CPrivate,
		},
		{
			name: "Public",
			args: args{
				ch: slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{IsIM: false, IsMpIM: false, IsPrivate: false}}},
			},
			want: CPublic,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChannelType(tt.args.ch); got != tt.want {
				t.Errorf("ChannelType() = %v, want %v", got, tt.want)
			}
		})
	}
}
