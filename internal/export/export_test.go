package export

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rusq/slackdump"
	"github.com/rusq/slackdump/internal/fixtures"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestExport_saveChannel(t *testing.T) {
	//TODO
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
		name              string
		fields            fields
		args              args
		wantErr           bool
		wantMessageByDate messagesByDate
	}{
		{
			"save ok",
			fields{dir: dir},
			args{
				"unittest",
				messagesByDate{
					"2020-12-30": []ExportMessage{
						{Msg: fixtures.Load[slack.Msg](fixtures.SimpleMessageJSON)},
					},
					"2020-12-31": []ExportMessage{
						{Msg: fixtures.Load[slack.Msg](fixtures.SimpleMessageJSON)},
						{Msg: fixtures.Load[slack.Msg](fixtures.BotMessageThreadParentJSON)},
						{Msg: fixtures.Load[slack.Msg](fixtures.BotMessageThreadChildJSON)},
					},
				},
			},
			false,
			messagesByDate{
				"2020-12-30": []ExportMessage{
					{Msg: fixtures.Load[slack.Msg](fixtures.SimpleMessageJSON)},
				},
				"2020-12-31": []ExportMessage{
					{Msg: fixtures.Load[slack.Msg](fixtures.SimpleMessageJSON)},
					{Msg: fixtures.Load[slack.Msg](fixtures.BotMessageThreadParentJSON)},
					{Msg: fixtures.Load[slack.Msg](fixtures.BotMessageThreadChildJSON)},
				},
			},
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
			mbd, err := loadTestDir(filepath.Join(dir, tt.args.channelName))
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, tt.wantMessageByDate, mbd)
		})
	}
}

func loadTestDir(path string) (messagesByDate, error) {
	// no proper error checking.
	var mbd = make(messagesByDate, 0)
	if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if filepath.Ext(path) != ".json" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		var mm []ExportMessage
		dec := json.NewDecoder(f)
		if err := dec.Decode(&mm); err != nil {
			return err
		}
		mbd[strings.TrimSuffix(filepath.Base(path), ".json")] = mm
		return nil
	}); err != nil {
		return nil, err
	}
	if err := mbd.validate(); err != nil {
		return nil, err
	}
	return mbd, nil
}
