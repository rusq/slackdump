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
package format

import (
	"bytes"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
	"github.com/stretchr/testify/assert"
)

var (
	testMsg1 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "d1831c57-3b7f-4a0c-ab9a-a18d4a58a01c",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497751.040300",
		Text:        "Test message \u0026lt; \u0026gt; \u0026lt; \u0026gt;",
	}}}
	testMsg2 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "b11431d3-a5c4-4612-b09c-b074e9ddace7",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497781.040300",
		Text:        "message 2",
	}}}
	testMsg3 = types.Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "a99df2f2-1fd6-421f-9453-6903974b683a",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1641541791.000000",
		Text:        "message 3",
	}}}
	testMsg4t = types.Message{
		Message: slack.Message{Msg: slack.Msg{
			ClientMsgID:     "931db474-6ea8-43bc-9ff7-804309716ded",
			Type:            "message",
			User:            "UP58RAHCJ",
			Timestamp:       "1638524854.042000",
			ThreadTimestamp: "1638524854.042000",
			ReplyCount:      3,
			Text:            "message 4",
		}},
		ThreadReplies: []types.Message{
			{Message: slack.Message{Msg: slack.Msg{
				ClientMsgID:     "a99df2f2-1fd6-421f-9453-6903974b683a",
				Type:            "message",
				Timestamp:       "1638554726.042700",
				ThreadTimestamp: "1638524854.042000",
				User:            "U01HPAR0YFN",
				Text:            "blah blah, reply 1",
			}}},
		},
	}
)

// test retrofitted from v2.
func TestText_Conversation(t *testing.T) {
	type args struct {
		m       []types.Message
		prefix  string
		userIdx structures.UserIndex
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			"two messages from the same person, not very far apart, with html escaped char",
			args{[]types.Message{testMsg1, testMsg2}, "", nil},
			"\n> <external>:U10H7D9RR [U10H7D9RR] @ 03/12/2021 02:15:51 Z:\nTest message < > < >\nmessage 2\n",
			false,
		},
		{
			"two messages from the same person, far apart",
			args{[]types.Message{testMsg1, testMsg4t}, "", nil},
			"\n> <external>:U10H7D9RR [U10H7D9RR] @ 03/12/2021 02:15:51 Z:\nTest message < > < >\n\n> <external>:UP58RAHCJ [UP58RAHCJ] @ 03/12/2021 09:47:34 Z:\nmessage 4\n|   \n|   > <external>:U01HPAR0YFN [U01HPAR0YFN] @ 03/12/2021 18:05:26 Z:\n|   blah blah, reply 1\n",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			txt := NewText()

			if err := txt.Conversation(t.Context(), buf, nil, &types.Conversation{Messages: tt.args.m}); (err != nil) != tt.wantErr {
				t.Errorf("Session.generateText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotW := buf.String()
			assert.Equal(t, tt.wantW, gotW)
		})
	}

}
