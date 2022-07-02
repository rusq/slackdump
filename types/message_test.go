package types

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
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
