package slackdump

import (
	"reflect"
	"sync"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

var (
	file1 = slack.File{ID: "f1", Name: "filename1.ext"}
	file2 = slack.File{ID: "f2", Name: "filename2.ext"}
	file3 = slack.File{ID: "f3", Name: "filename3.ext"}
	file4 = slack.File{ID: "f4", Name: "filename4.ext"}
	file5 = slack.File{ID: "f5", Name: "filename5.ext"}
	file6 = slack.File{ID: "f6", Name: "filename6.ext"}

	testFileMsg1 = Message{
		Message: slack.Message{
			Msg: slack.Msg{
				ClientMsgID: "1",
				Channel:     "x",
				Type:        "y",
				Files: []slack.File{
					file1, file2, file3,
				}},
		}}
	testFileMsg2 = Message{
		Message: slack.Message{
			Msg: slack.Msg{
				ClientMsgID: "2",
				Channel:     "x",
				Type:        "z",
				Files: []slack.File{
					file4, file5, file6,
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
				file1, file2, file3, file4, file5, file6,
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

func TestSlackDumper_pipeFiles(t *testing.T) {
	sd := SlackDumper{
		options: options{
			dumpfiles: true,
		},
	}

	want := []slack.File{
		file1, file2, file3, file4, file5, file6,
	}

	var wg sync.WaitGroup

	var got []slack.File
	filesC := make(chan *slack.File)
	go func(c <-chan *slack.File) {
		// catcher
		for f := range c {
			got = append(got, *f)
		}
		wg.Done()
	}(filesC)
	wg.Add(1)

	sd.pipeFiles(filesC, []Message{testFileMsg1, testFileMsg2})
	close(filesC)
	wg.Wait()

	assert.Equal(t, want, got)
}
