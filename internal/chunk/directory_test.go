package chunk

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/fixtures"
)

// assortment of channel info chunks
var (
	TestPublicChannelInfo = Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:       "C01SPFM1KNY",
					IsShared: false,
				},
				Name:       "test",
				IsArchived: false,
			},
			IsChannel: true,
			IsMember:  true,
			IsGeneral: false,
		},
	}
	TestDMChannelInfo = Chunk{
		ChannelID: "D01MN4X7UGP",
		Type:      CChannelInfo,
		Channel: &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Conversation: slack.Conversation{
					ID:          "D01MN4X7UGP",
					IsOpen:      true,
					IsIM:        true,
					IsPrivate:   true,
					IsOrgShared: false,
				},
			},
		},
	}
	TestChannelUsers = Chunk{
		ChannelID: "C01SPFM1KNY",
		Type:      CChannelUsers,
		ChannelUsers: []string{
			"U01SPFM1KNY",
			"U01SPFM1KNZ",
			"U01SPFM1KNA",
		},
	}
)

// assortment of message chunks
var (
	TestPublicChannelMessages = Chunk{
		Type:      CMessages,
		ChannelID: "C01SPFM1KNY",
		Messages: []slack.Message{
			fixtures.Load[slack.Message](fixtures.TestMessage),
		},
	}
)

func TestOpenDir(t *testing.T) {
}

func TestDirectory_version(t *testing.T) {
	type fields struct {
		dir        string
		cache      dcache
		fm         *filemgr
		numWorkers int
		timestamp  int64
		wantCache  bool
		readOnly   bool
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name:    "test",
			fields:  fields{},
			args:    args{name: "channels.json.gz"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "some version",
			fields:  fields{},
			args:    args{name: "channels_123.json.gz"},
			want:    123,
			wantErr: false,
		},
		{
			name:    "parse error",
			fields:  fields{},
			args:    args{name: "channels_abc.json.gz"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Directory{
				dir:        tt.fields.dir,
				cache:      tt.fields.cache,
				fm:         tt.fields.fm,
				numWorkers: tt.fields.numWorkers,
				timestamp:  tt.fields.timestamp,
				wantCache:  tt.fields.wantCache,
				readOnly:   tt.fields.readOnly,
			}
			got, err := d.version(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.version() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Directory.version() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectory_versions(t *testing.T) {
	type fields struct {
		dir        string
		cache      dcache
		fm         *filemgr
		numWorkers int
		timestamp  int64
		wantCache  bool
		readOnly   bool
	}
	type args struct {
		names []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []int64
		wantErr bool
	}{
		{
			name:   "single file",
			fields: fields{},
			args: args{
				names: []string{"channels.json.gz"},
			},
			want:    []int64{0},
			wantErr: false,
		},
		{
			name:   "multiple files",
			fields: fields{},
			args: args{
				names: []string{"channels.json.gz", "channels_123.json.gz", "channels_456.json.gz"},
			},
			want:    []int64{0, 123, 456},
			wantErr: false,
		},
		{
			name:   "parse error",
			fields: fields{},
			args: args{
				names: []string{"channels.json.gz", "channels_abc.json.gz", "channels_456.json.gz"},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Directory{
				dir:        tt.fields.dir,
				cache:      tt.fields.cache,
				fm:         tt.fields.fm,
				numWorkers: tt.fields.numWorkers,
				timestamp:  tt.fields.timestamp,
				wantCache:  tt.fields.wantCache,
				readOnly:   tt.fields.readOnly,
			}
			got, err := d.versions(tt.args.names...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Directory.versions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Directory.versions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirectory_filever(t *testing.T) {
	type fields struct {
		dir        string
		cache      dcache
		fm         *filemgr
		numWorkers int
		timestamp  int64
		wantCache  bool
		readOnly   bool
	}
	type args struct {
		id  FileID
		ver int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name:   "test",
			fields: fields{dir: "testdata"},
			args: args{
				id:  FChannels,
				ver: 123,
			},
			want: filepath.Join("testdata", "channels_123.json.gz"),
		},
		{
			name:   "base version",
			fields: fields{dir: "testdata"},
			args: args{
				id:  FChannels,
				ver: 0,
			},
			want: filepath.Join("testdata", "channels.json.gz"),
		},
		{
			name:   "mask",
			fields: fields{dir: "testdata"},
			args: args{
				id:  FChannels,
				ver: -1,
			},
			want: filepath.Join("testdata", "channels_*.json.gz"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Directory{
				dir:        tt.fields.dir,
				cache:      tt.fields.cache,
				fm:         tt.fields.fm,
				numWorkers: tt.fields.numWorkers,
				timestamp:  tt.fields.timestamp,
				wantCache:  tt.fields.wantCache,
				readOnly:   tt.fields.readOnly,
			}
			if got := d.filever(tt.args.id, tt.args.ver); got != tt.want {
				t.Errorf("Directory.filever() = %v, want %v", got, tt.want)
			}
		})
	}
}
