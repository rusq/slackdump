// Package control holds the implementation of the Slack Stream controller.
// It runs the API scraping in several goroutines and manages the data flow
// between them.  It records the output of the API scraper into a chunk
// directory.  It also manages the transformation of the data, if the caller
// is interested in it.
package control

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/rusq/slackdump/v3/processor"
)

var testUsers = []slack.User{
	testUser1,
	testUser2,
}

func TestDirController_Run(t *testing.T) {
	type fields struct {
		// cd *chunk.Directory
		// s       Streamer
		options options
	}
	type args struct {
		ctx  context.Context
		list *structures.EntityList
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors, list provided",
			fields: fields{
				options: options{
					lg:    slog.Default(),
					tf:    &noopTransformer{},
					filer: &noopFiler{},
					avp:   &noopAvatarProc{},
				},
			},
			args: args{
				ctx: context.Background(),
				list: structures.NewEntityListFromItems(structures.EntityItem{
					Id:      testPubChanMember.ID,
					Include: true,
				}),
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, proc processor.Users, opts ...slack.GetUsersOption) error {
						return proc.Users(ctx, testUsers)
					},
				)
			},
			wantErr: false,
		},
		{
			name: "no errors, list not given",
			fields: fields{
				options: options{
					lg:    slog.Default(),
					tf:    &noopTransformer{},
					filer: &noopFiler{},
					avp:   &noopAvatarProc{},
				},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil) // all channels are included, so it should get them.
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, proc processor.Users, opts ...slack.GetUsersOption) error {
						return proc.Users(ctx, testUsers)
					},
				)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(s)
			}

			dir := t.TempDir()
			cd, err := chunk.OpenDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer cd.Close()

			c := &DirController{
				cd:      cd,
				s:       s,
				options: tt.fields.options,
			}
			if err := c.Run(tt.args.ctx, tt.args.list); (err != nil) != tt.wantErr {
				t.Errorf("DirController.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDir(t *testing.T) {
	testDir := t.TempDir()
	cd, err := chunk.OpenDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer cd.Close()
	type args struct {
		cd   *chunk.Directory
		s    Streamer
		opts []Option
	}
	tests := []struct {
		name string
		args args
		want *DirController
	}{
		{
			name: "creates new controller",
			args: args{
				cd: cd,
				s:  &mock_control.MockStreamer{},
			},
			want: &DirController{
				cd: cd,
				s:  &mock_control.MockStreamer{},
				options: options{
					lg:    slog.Default(),
					tf:    &noopTransformer{},
					filer: &noopFiler{},
					avp:   &noopAvatarProc{},
				},
			},
		},
		{
			name: "options get processed",
			args: args{
				cd: cd,
				s:  &mock_control.MockStreamer{},
				opts: []Option{
					WithFiler(&mock_processor.MockFiler{}),
					WithAvatarProcessor(&mock_processor.MockAvatars{}),
				},
			},
			want: &DirController{
				cd: cd,
				s:  &mock_control.MockStreamer{},
				options: options{
					lg:    slog.Default(),
					tf:    &noopTransformer{},
					filer: &mock_processor.MockFiler{},
					avp:   &mock_processor.MockAvatars{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDir(tt.args.cd, tt.args.s, tt.args.opts...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirController_Close(t *testing.T) {
	type fields struct {
		cd *chunk.Directory
		s  Streamer
		// options options
	}
	tests := []struct {
		name     string
		fields   fields
		expectFn func(*mock_processor.MockFiler, *mock_processor.MockAvatars)
		wantErr  bool
	}{
		{
			name: "no errors",
			fields: fields{
				cd: nil,
				s:  &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars) {
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				cd: nil,
				s:  &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars) {
				f.EXPECT().Close().Return(assert.AnError)
				a.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "error",
			fields: fields{
				cd: nil,
				s:  &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars) {
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			f := mock_processor.NewMockFiler(ctrl)
			a := mock_processor.NewMockAvatars(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(f, a)
			}
			c := &DirController{
				cd: tt.fields.cd,
				s:  tt.fields.s,
				options: options{
					lg:    slog.Default(),
					tf:    &noopTransformer{},
					filer: f,
					avp:   a,
				},
			}
			if err := c.Close(); (err != nil) != tt.wantErr {
				t.Errorf("DirController.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
