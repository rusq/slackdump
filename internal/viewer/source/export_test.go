package source

import (
	"archive/zip"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

var testZipFile = filepath.Join("..", "..", "..", "tmp", "realexport.zip")

func openTestZip(t *testing.T, name string) *zip.ReadCloser {
	fixtures.SkipInCI(t)
	fixtures.SkipIfNotExist(t, name)

	t.Helper()
	zr, err := zip.OpenReader(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { zr.Close() })
	return zr
}

func TestExport_Channels(t *testing.T) {
	type fields struct {
		z         fs.FS
		chanNames map[string]string
		name      string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []slack.Channel
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				z: openTestZip(t, testZipFile),
			},
			want:    fixtures.Load[[]slack.Channel](fixtures.TestChannelsNativeExport),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Export{
				fs:        tt.fields.z,
				chanNames: tt.fields.chanNames,
				name:      tt.fields.name,
			}
			got, err := e.Channels()
			if (err != nil) != tt.wantErr {
				t.Errorf("Export.Channels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExport_AllMessages(t *testing.T) {
	type fields struct {
		z         fs.FS
		chanNames map[string]string
		name      string
	}
	type args struct {
		channelID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []slack.Message
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				z: openTestZip(t, testZipFile),
				chanNames: map[string]string{
					"CHY5HUESG": "everyone",
				},
			},
			args: args{
				channelID: "CHY5HUESG",
			},
			want:    fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Export{
				fs:        tt.fields.z,
				chanNames: tt.fields.chanNames,
				name:      tt.fields.name,
			}
			got, err := e.AllMessages(tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ZIPExport.AllMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ZIPExport.AllMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildFileIndex(t *testing.T) {
	testpath := filepath.Join("..", "..", "..", "tmp", "stdexport")
	fixtures.SkipIfNotExist(t, testpath)

	type args struct {
		fsys fs.FS
		dir  string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				fsys: os.DirFS(testpath),
				dir:  ".",
			},
			want:    map[string]string{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFileIndex(tt.args.fsys, tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildFileIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
