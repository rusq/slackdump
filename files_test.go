package slackdump

import (
	"reflect"
	"testing"

	"github.com/slack-go/slack"
)

var (
	testFileMsg1 = Message{
		Message: slack.Message{
			Msg: slack.Msg{
				ClientMsgID: "1",
				Channel:     "x",
				Type:        "y",
				Files: []slack.File{
					{ID: "f1", Name: "filename1.ext"},
					{ID: "f2", Name: "filename2.ext"},
					{ID: "f3", Name: "filename3.ext"},
				}},
		}}
	testFileMsg2 = Message{
		Message: slack.Message{
			Msg: slack.Msg{
				ClientMsgID: "2",
				Channel:     "x",
				Type:        "z",
				Files: []slack.File{
					{ID: "f4", Name: "filename4.ext"},
					{ID: "f5", Name: "filename5.ext"},
					{ID: "f6", Name: "filename6.ext"},
				}},
		}}
)

func TestSlackDumper_filesFromMessages(t *testing.T) {
	type args struct {
		m []Message
	}
	tests := []struct {
		name string
		args args
		want []slack.File
	}{
		{
			"extracts files ok",
			args{[]Message{testFileMsg1, testFileMsg2}},
			[]slack.File{
				{ID: "f1", Name: "filename1.ext"},
				{ID: "f2", Name: "filename2.ext"},
				{ID: "f3", Name: "filename3.ext"},
				{ID: "f4", Name: "filename4.ext"},
				{ID: "f5", Name: "filename5.ext"},
				{ID: "f6", Name: "filename6.ext"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{}
			if got := sd.filesFromMessages(tt.args.m); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackDumper.filesFromMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}
