package source

import (
	"archive/zip"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
)

var testZipFile = filepath.Join("..", "..", "..", "tmp", "realexport.zip")

func openTestZip(t *testing.T, name string) *zip.ReadCloser {
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Export.Channels() = %v, want %v", got, tt.want)
			}
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
