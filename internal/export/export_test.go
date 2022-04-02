package export

import (
	"os/exec"
	"testing"

	"github.com/rusq/slackdump"
	"github.com/rusq/slackdump/internal/fixtures"
	"github.com/slack-go/slack"
)

func TestExport_saveChannel(t *testing.T) {
	dir := t.TempDir()
	type fields struct {
		dir    string
		dumper *slackdump.SlackDumper
	}
	type args struct {
		channelName string
		msgs        messagesByDate
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"save ok",
			fields{dir: dir},
			args{
				"unittest",
				messagesByDate{
					"2020-12-31": []ExportMessage{
						{Msg: fixtures.Load[slack.Msg](fixtures.SimpleMessageJSON)},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &Export{
				dir:    tt.fields.dir,
				dumper: tt.fields.dumper,
			}
			if err := se.saveChannel(tt.args.channelName, tt.args.msgs); (err != nil) != tt.wantErr {
				t.Errorf("Export.saveChannel() error = %v, wantErr %v", err, tt.wantErr)
			}
			cmd := exec.Command("ls", "-lR", dir)
			data, err := cmd.CombinedOutput()
			if err != nil {
				t.Error(err)
			}
			t.Log(string(data))
		})
	}
}
