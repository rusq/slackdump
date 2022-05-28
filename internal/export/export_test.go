package export

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/fixtures"
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
				fs:     fsadapter.NewDirectory(tt.fields.dir),
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

// loadTestDir loads the file from the directory uses the
// filenames (minus JSON suffix) as a key in messageByDate map
// and file contents as []ExportMessage value for the key.
func loadTestDir(path string) (messagesByDate, error) {
	const jsonExt = ".json"
	// no proper error checking.
	var mbd = make(messagesByDate, 0)
	if err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) != jsonExt {
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

		mbd[strings.TrimSuffix(filepath.Base(path), jsonExt)] = mm

		return nil

	}); err != nil {
		return nil, err
	}

	if err := mbd.validate(); err != nil {
		return nil, err
	}

	return mbd, nil
}

func Test_populateNames(t *testing.T) {
	type args struct {
		ch  []slack.Channel
		usr []slack.User
	}
	tests := []struct {
		name   string
		args   args
		wantCh []slack.Channel
	}{
		{
			"populates im, but not channels",
			args{
				ch: []slack.Channel{
					{GroupConversation: slack.GroupConversation{
						Name:         "general",
						Conversation: slack.Conversation{ID: "C123", NameNormalized: "general"},
					}},
					{GroupConversation: slack.GroupConversation{
						Name:         "",
						Conversation: slack.Conversation{ID: "C123", NameNormalized: "", User: "UABC", IsIM: true},
					}},
					{GroupConversation: slack.GroupConversation{
						Name:         "",
						Conversation: slack.Conversation{ID: "C234", NameNormalized: "", User: "UBCD", IsIM: true},
					}},
				},
				usr: []slack.User{
					{ID: "UABC", Name: "alice"},
					{ID: "UBCD", Name: "bob"},
				},
			},
			[]slack.Channel{
				{GroupConversation: slack.GroupConversation{
					Name:         "general",
					Conversation: slack.Conversation{ID: "C123", NameNormalized: "general"},
				}},
				{GroupConversation: slack.GroupConversation{
					Name:         "alice",
					Conversation: slack.Conversation{ID: "C123", NameNormalized: "alice", User: "UABC", IsIM: true},
				}},
				{GroupConversation: slack.GroupConversation{
					Name:         "bob",
					Conversation: slack.Conversation{ID: "C234", NameNormalized: "bob", User: "UBCD", IsIM: true},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			populateNames(tt.args.ch, tt.args.usr)
		})
		assert.Equal(t, tt.wantCh, tt.args.ch)
	}
}
