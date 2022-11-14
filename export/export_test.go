package export

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
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
		dumper *slackdump.Session
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
				fs: fsadapter.NewDirectory(tt.fields.dir),
				sd: tt.fields.dumper,
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

func Test_validName(t *testing.T) {
	type args struct {
		ch slack.Channel
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"im",
			args{slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{IsIM: true, ID: "ID42"}}}},
			"ID42",
		},
		{
			"channel (#144)",
			args{slack.Channel{GroupConversation: slack.GroupConversation{Name: "name", Conversation: slack.Conversation{IsIM: false, ID: "ID42", NameNormalized: "name_normalized"}}}},
			"name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validName(tt.args.ch)
			if got != tt.want {
				t.Errorf("validName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serializeToFS(t *testing.T) {
	const (
		testData = "123"
		want     = `"` + testData + `"` + "\n"
	)
	t.Run("directory", func(t *testing.T) {
		tempdir := t.TempDir()
		fsys := fsadapter.NewDirectory(tempdir)
		if err := serializeToFS(fsys, "test.json", testData); err != nil {
			t.Fatal(err)
		}
		// read back
		got, err := os.ReadFile(filepath.Join(tempdir, "test.json"))
		if err != nil {
			t.Fatal(err)
		}

		if !strings.EqualFold(string(got), want) {
			t.Errorf("data mismatch: want=%q, got=%q", want, string(got))
		}
	})
	t.Run("zipFile", func(t *testing.T) {
		tempdir := t.TempDir()
		testzip := filepath.Join(tempdir, "test.zip")
		fsys, err := fsadapter.NewZipFile(testzip)
		if err != nil {
			t.Fatal(err)
		}
		if err := serializeToFS(fsys, "test.json", testData); err != nil {
			t.Fatal(err)
		}
		fsys.Close()

		// read back
		arc, err := zip.OpenReader(testzip)
		if err != nil {
			t.Fatal(err)
		}
		defer arc.Close()

		r, err := arc.Open("test.json")
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.EqualFold(string(got), want) {
			t.Errorf("data mismatch: want=%q, got=%q", want, string(got))
		}
	})
	t.Run("fs error", func(t *testing.T) {
		if err := serializeToFS(errFs{}, "test.fs", testData); err == nil {
			t.Fatal("expected error, but got nil")
		}
	})
}

type errFs struct{}

func (errFs) Create(string) (io.WriteCloser, error) {
	return nil, errors.New("not this time")
}

func (errFs) WriteFile(name string, data []byte, perm os.FileMode) error {
	return errors.New("no luck bro")
}

func (errFs) Close() error {
	return errors.New("close error")
}
