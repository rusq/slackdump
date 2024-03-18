package structures

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
	"testing/fstest"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/fsadapter/mocks/mock_fsadapter"
	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_io"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestExportIndex_mostFrequentMember(t *testing.T) {
	type args struct {
		dms []DM
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"finds me",
			args{[]DM{{Members: []string{"me"}}}},
			"me",
		},
		{
			"finds me in several dms",
			args{[]DM{{Members: []string{"me", "you"}}, {Members: []string{"me", "someone_else"}}, {Members: []string{"me"}}}},
			"me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mostFrequentMember(tt.args.dms); got != tt.want {
				t.Errorf("ExportIndex.me() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExportIndex_except(t *testing.T) {
	type args struct {
		me      string
		members []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"finds not me",
			args{"me", []string{"you", "me"}},
			"you",
		},
		{
			"finds not me in several members",
			args{"me", []string{"you", "me", "someone_else"}},
			"you",
		},
		{
			"returns empty string if no not me",
			args{"me", []string{"me"}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := except(tt.args.me, tt.args.members); got != tt.want {
				t.Errorf("ExportIndex.notMe() = %v, want %v", got, tt.want)
			}
		})
	}
}

type teststruct struct {
	String string `json:"string"`
}

func Test_unmarshalFileFS(t *testing.T) {
	sys := fstest.MapFS{
		"filename": &fstest.MapFile{
			Data:    []byte(`{"string":"passed"}`),
			Mode:    0644,
			ModTime: time.Now(),
			Sys:     nil,
		},
	}
	type args struct {
		fsys     fs.FS
		filename string
		data     any
	}
	tests := []struct {
		name    string
		args    args
		wantAny any
		wantErr bool
	}{
		{
			"loads from fs",
			args{sys, "filename", &teststruct{}},
			&teststruct{String: "passed"},
			false,
		},
		{
			"file does not exist",
			args{sys, "nonexistent", &teststruct{}},
			&teststruct{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := unmarshalFileFS(tt.args.fsys, tt.args.filename, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("unmarshalFileFS() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.data, tt.wantAny) {
				t.Errorf("unmarshalFileFS() got = %v, want %v", tt.args.data, tt.wantAny)
			}
		})
	}
}

func Test_marshalFileFSA(t *testing.T) {
	type args struct {
		// fs       fsadapter.FS
		filename string
		data     any
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_fsadapter.MockFSCloser, *mock_io.MockWriteCloser)
		wantErr  bool
	}{
		{
			"writes to fs",
			args{"filename", teststruct{String: "passed"}},
			func(m *mock_fsadapter.MockFSCloser, wc *mock_io.MockWriteCloser) {
				wc.EXPECT().Write(gomock.Any()).Return(len("passed"), nil)
				wc.EXPECT().Close().Return(nil)
				m.EXPECT().Create("filename").Return(wc, nil)
			},
			false,
		},
		{
			"create error",
			args{"filename", teststruct{String: "passed"}},
			func(m *mock_fsadapter.MockFSCloser, wc *mock_io.MockWriteCloser) {
				m.EXPECT().Create("filename").Return(nil, &fs.PathError{})
			},
			true,
		},
		{
			"write error",
			args{"filename", teststruct{String: "passed"}},
			func(m *mock_fsadapter.MockFSCloser, wc *mock_io.MockWriteCloser) {
				wc.EXPECT().Close().Return(nil)
				wc.EXPECT().Write(gomock.Any()).Return(0, os.ErrClosed)
				m.EXPECT().Create("filename").Return(wc, nil)
			},
			true,
		},
		{
			"close error",
			args{"filename", teststruct{String: "passed"}},
			func(m *mock_fsadapter.MockFSCloser, wc *mock_io.MockWriteCloser) {
				wc.EXPECT().Close().Return(os.ErrInvalid)
				wc.EXPECT().Write(gomock.Any()).Return(len("passed"), nil)
				m.EXPECT().Create("filename").Return(wc, nil)
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfc := mock_fsadapter.NewMockFSCloser(ctrl)
			mwc := mock_io.NewMockWriteCloser(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mfc, mwc)
			}
			if err := marshalFileFSA(mfc, tt.args.filename, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("marshalFileFSA() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExportIndex_Restore(t *testing.T) {
	type fields struct {
		Channels []slack.Channel
		Groups   []slack.Channel
		MPIMs    []slack.Channel
		DMs      []DM
		Users    []slack.User
	}
	tests := []struct {
		name   string
		fields fields
		want   []slack.Channel
	}{
		{
			"restores index to channels",
			fields{
				Channels: []slack.Channel{
					{
						GroupConversation: slack.GroupConversation{
							Conversation: slack.Conversation{ID: "C01"},
							Name:         "channel",
						},
					},
				},
				Groups: nil,
				MPIMs:  nil,
				DMs: []DM{
					{ID: "D01", Members: []string{"me", "you"}, Created: 1234567890},
					{ID: "D02", Members: []string{"me", "not you"}, Created: 1234567890},
				},
			},
			[]slack.Channel{
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{ID: "C01"},
						Name:         "channel",
					},
				},
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID:      "D01",
							User:    "you",
							Created: slack.JSONTime(1234567890),
							IsIM:    true,
						},
						Members: []string{"me", "you"},
					},
				},
				{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID:      "D02",
							User:    "not you",
							Created: slack.JSONTime(1234567890),
							IsIM:    true,
						},
						Members: []string{"me", "not you"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := &ExportIndex{
				Channels: tt.fields.Channels,
				Groups:   tt.fields.Groups,
				MPIMs:    tt.fields.MPIMs,
				DMs:      tt.fields.DMs,
				Users:    tt.fields.Users,
			}
			got := idx.Restore()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExportIndex_Unmarshal(t *testing.T) {
	sampleTime := time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC)
	sys := fstest.MapFS{
		"channels.json": &fstest.MapFile{
			Data:    fixtures.TestExpChannelsJSON,
			Mode:    0644,
			ModTime: sampleTime,
		},
		"groups.json": &fstest.MapFile{
			Data:    fixtures.TestExpGroupsJSON,
			Mode:    0644,
			ModTime: sampleTime,
		},
		"mpims.json": &fstest.MapFile{
			Data:    fixtures.TestExpMPIMsJSON,
			Mode:    0644,
			ModTime: sampleTime,
		},
		"dms.json": &fstest.MapFile{
			Data:    fixtures.TestExpDMsJSON,
			Mode:    0644,
			ModTime: sampleTime,
		},
		"users.json": &fstest.MapFile{
			Data:    fixtures.TestExpUsersJSON,
			Mode:    0644,
			ModTime: sampleTime,
		},
	}
	type fields struct {
		Channels []slack.Channel
		Groups   []slack.Channel
		MPIMs    []slack.Channel
		DMs      []DM
		Users    []slack.User
	}
	type args struct {
		fsys fs.FS
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"unmarshals from fs",
			fields{},
			args{sys},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := &ExportIndex{
				Channels: tt.fields.Channels,
				Groups:   tt.fields.Groups,
				MPIMs:    tt.fields.MPIMs,
				DMs:      tt.fields.DMs,
				Users:    tt.fields.Users,
			}
			if err := idx.Unmarshal(tt.args.fsys); (err != nil) != tt.wantErr {
				t.Errorf("ExportIndex.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExportIndex_Marshal(t *testing.T) {
	type fields struct {
		Channels []slack.Channel
		Groups   []slack.Channel
		MPIMs    []slack.Channel
		DMs      []DM
		Users    []slack.User
	}
	type args struct {
		fs fsadapter.FS
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := &ExportIndex{
				Channels: tt.fields.Channels,
				Groups:   tt.fields.Groups,
				MPIMs:    tt.fields.MPIMs,
				DMs:      tt.fields.DMs,
				Users:    tt.fields.Users,
			}
			if err := idx.Marshal(tt.args.fs); (err != nil) != tt.wantErr {
				t.Errorf("ExportIndex.Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMakeExportIndex(t *testing.T) {
	type args struct {
		channels      []slack.Channel
		users         []slack.User
		currentUserID string
	}
	tests := []struct {
		name    string
		args    args
		want    *ExportIndex
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeExportIndex(tt.args.channels, tt.args.users, tt.args.currentUserID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeExportIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeExportIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
