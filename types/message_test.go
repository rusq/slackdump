package types

import (
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

var (
	testMsg1 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "d1831c57-3b7f-4a0c-ab9a-a18d4a58a01c",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497751.040300",
		Text:        "Test message \u0026lt; \u0026gt; \u0026lt; \u0026gt;",
	}}}
	testMsg2 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "b11431d3-a5c4-4612-b09c-b074e9ddace7",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1638497781.040300",
		Text:        "message 2",
	}}}
	testMsg3 = Message{Message: slack.Message{Msg: slack.Msg{
		ClientMsgID: "a99df2f2-1fd6-421f-9453-6903974b683a",
		Type:        "message",
		User:        "U10H7D9RR",
		Timestamp:   "1641541791.000000",
		Text:        "message 3",
	}}}
	testMsg4t = Message{
		Message: slack.Message{Msg: slack.Msg{
			ClientMsgID:     "931db474-6ea8-43bc-9ff7-804309716ded",
			Type:            "message",
			User:            "UP58RAHCJ",
			Timestamp:       "1638524854.042000",
			ThreadTimestamp: "1638524854.042000",
			ReplyCount:      3,
			Text:            "message 4",
		}},
		ThreadReplies: []Message{
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

func Test_sortMessages(t *testing.T) {
	type args struct {
		msgs []Message
	}
	tests := []struct {
		name     string
		args     args
		wantMsgs []Message
	}{
		{
			"empty",
			args{[]Message{}},
			[]Message{},
		},
		{
			"sort ok",
			args{[]Message{
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425514",
				}}},
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425511",
				}}},
			}},
			[]Message{
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425511",
				}}},
				{Message: slack.Message{Msg: slack.Msg{
					Timestamp: "1643425514",
				}}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortMessages(tt.args.msgs)
			assert.Equal(t, tt.wantMsgs, tt.args.msgs)
		})
	}
}

func TestSession_convertMsgs(t *testing.T) {
	type args struct {
		sm []slack.Message
	}
	tests := []struct {
		name string
		args args
		want []Message
	}{
		{
			"ok",
			args{[]slack.Message{
				testMsg1.Message,
				testMsg2.Message,
				testMsg3.Message,
			}},
			[]Message{
				testMsg1,
				testMsg2,
				testMsg3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertMsgs(tt.args.sm); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Session.convertMsgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
