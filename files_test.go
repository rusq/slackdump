package slackdump

import (
	"context"
	"os"
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

var (
	file1 = slack.File{ID: "f1", Name: "filename1.ext", URLPrivateDownload: "file1_url", Size: 100}
	file2 = slack.File{ID: "f2", Name: "filename2.ext", URLPrivateDownload: "file2_url", Size: 200}
	file3 = slack.File{ID: "f3", Name: "filename3.ext", URLPrivateDownload: "file3_url", Size: 300}
	file4 = slack.File{ID: "f4", Name: "filename4.ext", URLPrivateDownload: "file4_url", Size: 400}
	file5 = slack.File{ID: "f5", Name: "filename5.ext", URLPrivateDownload: "file5_url", Size: 500}
	file6 = slack.File{ID: "f6", Name: "filename6.ext", URLPrivateDownload: "file6_url", Size: 600}
	file7 = slack.File{ID: "f7", Name: "filename7.ext", URLPrivateDownload: "file7_url", Size: 700}
	file8 = slack.File{ID: "f8", Name: "filename8.ext", URLPrivateDownload: "file8_url", Size: 800}
	file9 = slack.File{ID: "f9", Name: "filename9.ext", URLPrivateDownload: "file9_url", Size: 900}

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

	testFileMsg3t = Message{
		Message: slack.Message{
			Msg: slack.Msg{
				ClientMsgID: "3",
				Channel:     "x",
				Type:        "z",
				Files: []slack.File{
					file7,
				}},
		},
		ThreadReplies: []Message{
			{
				Message: slack.Message{
					Msg: slack.Msg{
						ClientMsgID: "4",
						Channel:     "x",
						Type:        "message",
						Files: []slack.File{
							file8, file9,
						}},
				},
			},
		},
	}
)

func TestSlackDumper_ExtractFiles(t *testing.T) {
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
		{
			"extracts files from thread",
			args{[]Message{testFileMsg3t}},
			[]slack.File{file7, file8, file9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SlackDumper{}
			got := sd.ExtractFiles(tt.args.m)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSlackDumper_pipeFiles(t *testing.T) {
	sd := SlackDumper{
		options: Options{
			DumpFiles: true,
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

func TestSlackDumper_SaveFileTo(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	type fields struct {
		Users     Users
		UserIndex map[string]*slack.User
		options   Options
	}
	type args struct {
		ctx context.Context
		dir string
		f   *slack.File
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mc *mockClienter)
		want     int64
		wantErr  bool
	}{
		{
			"ok",
			fields{options: DefOptions},
			args{
				context.Background(),
				tmpdir,
				&file1,
			},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetFile("file1_url", gomock.Any()).
					Return(nil)
			},
			int64(file1.Size),
			false,
		},
		{
			"getfile rekt",
			fields{options: DefOptions},
			args{
				context.Background(),
				tmpdir,
				&file2,
			},
			func(mc *mockClienter) {
				mc.EXPECT().
					GetFile("file2_url", gomock.Any()).
					Return(errors.New("rekt"))
			},
			int64(0),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := newmockClienter(ctrl)

			tt.expectFn(mc)

			sd := &SlackDumper{
				client:    mc,
				Users:     tt.fields.Users,
				UserIndex: tt.fields.UserIndex,
				options:   tt.fields.options,
			}
			got, err := sd.SaveFileTo(tt.args.ctx, tt.args.dir, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackDumper.SaveFileTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackDumper.SaveFileTo() = %v, want %v", got, tt.want)
			}
		})
	}
}
