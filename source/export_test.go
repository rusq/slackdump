package source

import (
	"archive/zip"
	"context"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/structures"
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
			got, err := e.Channels(t.Context())
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
		ctx       context.Context
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
				ctx:       t.Context(),
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
			got, err := e.AllMessages(tt.args.ctx, tt.args.channelID)
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

func Test_buildStdFileIdx(t *testing.T) {
	testpath := filepath.Join("..", "..", "..", "tmp", "stdexport")
	fixtures.SkipIfNotExist(t, testpath)
	fixtures.SkipInCI(t)

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
			got, err := buildStdFileIdx(tt.args.fsys, tt.args.dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildStdFileIdx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExportChanName(t *testing.T) {
	var pub, dm slack.Channel
	pub.ID = "C123456"
	pub.Name = "general"

	dm.ID = "D123456"
	dm.IsIM = true

	type args struct {
		ch *slack.Channel
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test",
			args: args{ch: &pub},
			want: "general",
		},
		{
			name: "dm",
			args: args{ch: &dm},
			want: "D123456",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExportChanName(tt.args.ch); got != tt.want {
				t.Errorf("ExportChanName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExport_WorkspaceInfo(t *testing.T) {
	type fields struct {
		fs        fs.FS
		channels  []slack.Channel
		chanNames map[string]string
		name      string
		idx       structures.ExportIndex
		files     Storage
		avatars   Storage
	}
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *slack.AuthTestResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Export{
				fs:        tt.fields.fs,
				channels:  tt.fields.channels,
				chanNames: tt.fields.chanNames,
				name:      tt.fields.name,
				idx:       tt.fields.idx,
				files:     tt.fields.files,
				avatars:   tt.fields.avatars,
			}
			got, err := e.WorkspaceInfo(tt.args.in0)
			if (err != nil) != tt.wantErr {
				t.Errorf("Export.WorkspaceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Export.WorkspaceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExport_Latest(t *testing.T) {
	type fields struct {
		fs        fs.FS
		channels  []slack.Channel
		chanNames map[string]string
		name      string
		idx       structures.ExportIndex
		files     Storage
		avatars   Storage
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[structures.SlackLink]time.Time
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Export{
				fs:        tt.fields.fs,
				channels:  tt.fields.channels,
				chanNames: tt.fields.chanNames,
				name:      tt.fields.name,
				idx:       tt.fields.idx,
				files:     tt.fields.files,
				avatars:   tt.fields.avatars,
			}
			got, err := e.Latest(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Export.Latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Export.Latest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadStorage(t *testing.T) {
	mattermostFS := fstest.MapFS{
		path.Join(chunk.UploadsDir, "F123456", "somefile.txt"): {
			Data: []byte("test"),
		},
	}
	mmOverMapFS, err := OpenMattermostStorage(mattermostFS)
	if err != nil {
		t.Fatal(err)
	}

	stdFS := fstest.MapFS{
		path.Join("random", "attachments", "F123456-somefile.txt"): {
			Data: []byte("test"),
		},
	}
	stdOverMapFS, err := OpenStandardStorage(stdFS)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		fsys fs.FS
	}
	tests := []struct {
		name    string
		args    args
		want    Storage
		wantErr bool
	}{
		{
			name: "mattermost",
			args: args{
				fsys: mattermostFS,
			},
			want:    mmOverMapFS,
			wantErr: false,
		},
		{
			name: "standard",
			args: args{
				fsys: stdFS,
			},
			want:    stdOverMapFS,
			wantErr: false,
		},
		{
			name: "not found",
			args: args{
				fsys: fstest.MapFS{},
			},
			want:    NoStorage{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadStorage(tt.args.fsys)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExport_walkChannelMessages(t *testing.T) {
	type fields struct {
		fs        fs.FS
		channels  []slack.Channel
		chanNames map[string]string
		name      string
		idx       structures.ExportIndex
		files     Storage
		avatars   Storage
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
			name: "invalid json file",
			fields: fields{
				chanNames: map[string]string{
					"C123456": "general",
				},
				fs: fstest.MapFS{
					"general/2023-01-01.json": {
						Data: []byte("invalid json"),
					},
					"general/2023-01-02.json": {
						Data: []byte(`[{"type":"message","text":"Hello, world!"}]`),
					},
				},
			},
			args: args{
				channelID: "C123456",
			},
			want: []slack.Message{
				{Msg: slack.Msg{Type: "message", Text: "Hello, world!"}},
			},
		},
		{
			name: "ignores nested directories",
			fields: fields{
				chanNames: map[string]string{
					"C123456": "general",
				},
				fs: fstest.MapFS{
					"general/2023-01-01.json": {
						Data: []byte(`[{"type":"message","text":"Hello, world!"}]`),
					},
					"general/nested/2023-01-02.json": {
						Data: []byte(`[{"type":"message","text":"Nested message"}]`),
					},
				},
			},
			args: args{
				channelID: "C123456",
			},
			want: []slack.Message{
				{Msg: slack.Msg{Type: "message", Text: "Hello, world!"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Export{
				fs:        tt.fields.fs,
				channels:  tt.fields.channels,
				chanNames: tt.fields.chanNames,
				name:      tt.fields.name,
				idx:       tt.fields.idx,
				files:     tt.fields.files,
				avatars:   tt.fields.avatars,
			}
			it, err := e.walkChannelMessages(tt.args.channelID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Export.walkChannelMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var got []slack.Message
			for m, err := range it {
				if (err != nil) != tt.wantErr {
					t.Errorf("Export.walkChannelMessages() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				got = append(got, m)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Export.walkChannelMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}
