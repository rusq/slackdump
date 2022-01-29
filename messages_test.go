package slackdump

import (
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
			sortMessages(tt.args.msgs)
			assert.Equal(t, tt.wantMsgs, tt.args.msgs)
		})
	}
}
