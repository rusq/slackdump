package fileproc

import (
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
)

func Test_dumpSubproc_PathUpdate(t *testing.T) {
	type args struct {
		channelID string
		threadTS  string
		mm        []slack.Message
	}
	tests := []struct {
		name    string
		args    args
		wantMM  []slack.Message
		wantErr bool
	}{
		{
			"just a channel",
			args{
				channelID: "C12345678",
				threadTS:  "",
				mm: []slack.Message{
					{
						Msg: slack.Msg{
							Files: []slack.File{
								{
									ID:                 "F12345678",
									Name:               "file.txt",
									URLPrivate:         "https://files.slack.com/files-pri/T12345678-F12345678/file.txt",
									URLPrivateDownload: "https://files.slack.com/files-pri/T12345678-F12345678/download/file.txt",
								},
							},
						},
					},
				},
			},
			[]slack.Message{
				{
					Msg: slack.Msg{
						Files: []slack.File{
							{
								ID:                 "F12345678",
								Name:               "file.txt",
								URLPrivate:         "C12345678/F12345678-file.txt",
								URLPrivateDownload: "C12345678/F12345678-file.txt",
							},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := FileProcessor{
				filepath: DumpFilepath,
			}
			if err := d.PathUpdateFunc(tt.args.channelID, tt.args.threadTS, tt.args.mm); (err != nil) != tt.wantErr {
				t.Errorf("dumpSubproc.PathUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantMM, tt.args.mm)
		})
	}
}
