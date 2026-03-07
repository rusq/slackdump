// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package control

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
	"github.com/rusq/slackdump/v4/processor"
	"github.com/rusq/slackdump/v4/stream"
)

func TestController_Close(t *testing.T) {
	type fields struct {
		// erc     EncodeReferenceCloser
		s       Streamer
		options options
	}
	tests := []struct {
		name     string
		fields   fields
		expectFn func(*mock_processor.MockFiler, *mock_processor.MockAvatars, *mock_control.MockEncodeReferenceCloser)
		wantErr  bool
	}{
		{
			name: "no errors",
			fields: fields{
				s: &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars, erc *mock_control.MockEncodeReferenceCloser) {
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)
				erc.EXPECT().Close().Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error closing filer, the rest should continue",
			fields: fields{
				s: &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars, erc *mock_control.MockEncodeReferenceCloser) {
				f.EXPECT().Close().Return(assert.AnError)
				a.EXPECT().Close().Return(nil)
				erc.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "error closing avatar processor, the rest should continue",
			fields: fields{
				s: &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars, erc *mock_control.MockEncodeReferenceCloser) {
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(assert.AnError)
				erc.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "error closing erc, the rest should continue",
			fields: fields{
				s: &mock_control.MockStreamer{},
			},
			expectFn: func(f *mock_processor.MockFiler, a *mock_processor.MockAvatars, erc *mock_control.MockEncodeReferenceCloser) {
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)
				erc.EXPECT().Close().Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctrl = gomock.NewController(t)
				f    = mock_processor.NewMockFiler(ctrl)
				a    = mock_processor.NewMockAvatars(ctrl)
				erc  = mock_control.NewMockEncodeReferenceCloser(ctrl)
			)
			if tt.expectFn != nil {
				tt.expectFn(f, a, erc)
			}
			c := &Controller{
				erc:     erc,
				s:       tt.fields.s,
				options: tt.fields.options,
			}
			c.options.filer = f
			c.options.avp = a

			if err := c.Close(); (err != nil) != tt.wantErr {
				t.Errorf("Controller.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_Run(t *testing.T) {
	type args struct {
		ctx  context.Context
		list *structures.EntityList
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(
			*mock_control.MockStreamer,
			*mock_processor.MockFiler,
			*mock_processor.MockAvatars,
			*mock_control.MockExportTransformer,
			*mock_control.MockEncodeReferenceCloser,
		)
		wantErr bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:  t.Context(),
				list: &structures.EntityList{},
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				testUsers := []slack.User{testUser1, testUser2}
				// called by the runner
				s.EXPECT().ListChannelsEx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stream.ErrOpNotSupported)
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error {
					proc.Users(ctx, testUsers)
					return nil
				})
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				// called by close
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)

				// Users calls avatars
				a.EXPECT().Users(gomock.Any(), testUsers).Return(nil)
				// encoder calls
				erc.EXPECT().Encode(gomock.Any(), gomock.Any()).Return(nil)
				// once users are processed, transformer should be started
				tf.EXPECT().StartWithUsers(gomock.Any(), testUsers).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Users returns error",
			args: args{
				ctx:  t.Context(),
				list: &structures.EntityList{},
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				// called by the runner
				s.EXPECT().ListChannelsEx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stream.ErrOpNotSupported)
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any()).Return(assert.AnError)
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				// called by close
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctrl = gomock.NewController(t)
				s    = mock_control.NewMockStreamer(ctrl)
				f    = mock_processor.NewMockFiler(ctrl)
				a    = mock_processor.NewMockAvatars(ctrl)
				tf   = mock_control.NewMockExportTransformer(ctrl)
				erc  = mock_control.NewMockEncodeReferenceCloser(ctrl)
			)
			if tt.expectFn != nil {
				tt.expectFn(s, f, a, tf, erc)
			}
			c := &Controller{
				erc: erc,
				s:   s,
				options: options{
					filer: f,
					avp:   a,
					tf:    tf,
				},
			}
			if err := c.Run(tt.args.ctx, tt.args.list); (err != nil) != tt.wantErr {
				t.Errorf("Controller.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_RunNoTransform(t *testing.T) {
	type args struct {
		ctx  context.Context
		list *structures.EntityList
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(
			*mock_control.MockStreamer,
			*mock_processor.MockFiler,
			*mock_processor.MockAvatars,
			*mock_control.MockExportTransformer,
			*mock_control.MockEncodeReferenceCloser,
		)
		wantErr bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:  t.Context(),
				list: &structures.EntityList{},
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				testUsers := []slack.User{testUser1, testUser2}
				// called by the runner
				s.EXPECT().ListChannelsEx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stream.ErrOpNotSupported)
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error {
					proc.Users(ctx, testUsers)
					return nil
				})
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				// called by close
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)

				// Users calls avatars and recorder
				a.EXPECT().Users(gomock.Any(), testUsers).Return(nil)
				// encoder gets at least one encode (from Users via recorder)
				erc.EXPECT().Encode(gomock.Any(), gomock.Any()).Return(nil)
				// transformer should only be started with users; no Transform expected
				tf.EXPECT().StartWithUsers(gomock.Any(), testUsers).Return(nil)
				// ensure no completion checks are performed in RunNoTransform
				erc.EXPECT().IsComplete(gomock.Any(), gomock.Any()).Times(0)
				erc.EXPECT().IsCompleteThread(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: false,
		},
		{
			name: "Users returns error",
			args: args{
				ctx:  t.Context(),
				list: &structures.EntityList{},
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				// called by the runner
				s.EXPECT().ListChannelsEx(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(stream.ErrOpNotSupported)
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Users(gomock.Any(), gomock.Any()).Return(assert.AnError)
				s.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				// called by close (even on error)
				f.EXPECT().Close().Return(nil)
				a.EXPECT().Close().Return(nil)
				// ensure no completion checks are performed in error path as well
				erc.EXPECT().IsComplete(gomock.Any(), gomock.Any()).Times(0)
				erc.EXPECT().IsCompleteThread(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctrl = gomock.NewController(t)
				s    = mock_control.NewMockStreamer(ctrl)
				f    = mock_processor.NewMockFiler(ctrl)
				a    = mock_processor.NewMockAvatars(ctrl)
				tf   = mock_control.NewMockExportTransformer(ctrl)
				erc  = mock_control.NewMockEncodeReferenceCloser(ctrl)
			)
			if tt.expectFn != nil {
				tt.expectFn(s, f, a, tf, erc)
			}
			c := &Controller{
				erc: erc,
				s:   s,
				options: options{
					lg:    slog.Default(),
					filer: f,
					avp:   a,
					tf:    tf,
				},
			}
			if err := c.RunNoTransform(tt.args.ctx, tt.args.list); (err != nil) != tt.wantErr {
				t.Errorf("Controller.RunNoTransform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {
	type args struct {
		ctx  context.Context
		s    Streamer
		erc  EncodeReferenceCloser
		opts []Option
	}
	tests := []struct {
		name    string
		args    args
		want    *Controller
		wantErr bool
	}{
		{
			name: "creates new controller",
			args: args{
				ctx: t.Context(),
				s:   &mock_control.MockStreamer{},
				erc: &mock_control.MockEncodeReferenceCloser{},
			},
			want: &Controller{
				erc: &mock_control.MockEncodeReferenceCloser{},
				s:   &mock_control.MockStreamer{},
				options: options{
					lg:    slog.Default(),
					tf:    &noopExpTransformer{},
					filer: &noopFiler{},
					avp:   &noopAvatarProc{},
				},
			},
			wantErr: false,
		},
		{
			name: "options get processed",
			args: args{
				ctx: t.Context(),
				s:   &mock_control.MockStreamer{},
				erc: &mock_control.MockEncodeReferenceCloser{},
				opts: []Option{
					WithAvatarProcessor(&mock_processor.MockAvatars{}),
					WithFiler(&mock_processor.MockFiler{}),
				},
			},
			want: &Controller{
				erc: &mock_control.MockEncodeReferenceCloser{},
				s:   &mock_control.MockStreamer{},
				options: options{
					lg:    slog.Default(),
					tf:    &noopExpTransformer{},
					filer: &mock_processor.MockFiler{},
					avp:   &mock_processor.MockAvatars{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.ctx, tt.args.s, tt.args.erc, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_newConvTransformer(t *testing.T) {
	type fields struct {
		erc     EncodeReferenceCloser
		s       Streamer
		options options
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *conversationTransformer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				erc:     tt.fields.erc,
				s:       tt.fields.s,
				options: tt.fields.options,
			}
			if got := c.newConvTransformer(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.newConvTransformer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_Search(t *testing.T) {
	type args struct {
		ctx   context.Context
		query string
		stype SearchType
	}
	tests := []struct {
		name     string
		expectFn func(
			*mock_control.MockStreamer,
			*mock_processor.MockFiler,
			*mock_processor.MockAvatars,
			*mock_control.MockExportTransformer,
			*mock_control.MockEncodeReferenceCloser,
		)
		args    args
		wantErr bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:   t.Context(),
				query: "test",
				stype: SMessages | SFiles,
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "test").Return(nil)
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error searching messages",
			args: args{
				ctx:   t.Context(),
				query: "test",
				stype: SMessages | SFiles,
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "test").Return(assert.AnError)
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "test").Return(nil)
			},
			wantErr: true,
		},
		{
			name: "error searching files",
			args: args{
				ctx:   t.Context(),
				query: "test",
				stype: SMessages | SFiles,
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "test").Return(nil)
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "test").Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error getting workspace info",
			args: args{
				ctx:   t.Context(),
				query: "test",
				stype: SMessages | SFiles,
			},
			expectFn: func(s *mock_control.MockStreamer, f *mock_processor.MockFiler, a *mock_processor.MockAvatars, tf *mock_control.MockExportTransformer, erc *mock_control.MockEncodeReferenceCloser) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(assert.AnError)
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "test").Return(nil)
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "test").Return(nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				ctrl = gomock.NewController(t)
				s    = mock_control.NewMockStreamer(ctrl)
				f    = mock_processor.NewMockFiler(ctrl)
				a    = mock_processor.NewMockAvatars(ctrl)
				tf   = mock_control.NewMockExportTransformer(ctrl)
				erc  = mock_control.NewMockEncodeReferenceCloser(ctrl)
			)
			if tt.expectFn != nil {
				tt.expectFn(s, f, a, tf, erc)
			}
			c := &Controller{
				erc: erc,
				s:   s,
				options: options{
					lg:    slog.Default(),
					filer: f,
					avp:   a,
					tf:    tf,
				},
			}
			if err := c.Search(tt.args.ctx, tt.args.query, tt.args.stype); (err != nil) != tt.wantErr {
				t.Errorf("Controller.Search() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
