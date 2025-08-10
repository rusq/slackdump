package control

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slackdump/v3/internal/convert/transform"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"

	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/rusq/slackdump/v3/processor"
)

func Test_userWorker(t *testing.T) {
	type args struct {
		ctx context.Context
		// s   Streamer
		up processor.Users
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors",
			args: args{
				ctx: t.Context(),
				up:  &mock_processor.MockUsers{},
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().Users(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error listing users",
			args: args{
				ctx: t.Context(),
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().Users(gomock.Any(), gomock.Any()).Return(errors.New("error listing users"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			tt.expectFn(s)
			if err := userWorker(tt.args.ctx, s, tt.args.up); (err != nil) != tt.wantErr {
				t.Errorf("userWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_conversationWorker(t *testing.T) {
	type args struct {
		ctx context.Context
		// s     Streamer
		proc  processor.Conversations
		links <-chan structures.EntityItem
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:   t.Context(),
				proc:  &mock_processor.MockConversations{},
				links: make(<-chan structures.EntityItem),
			},
			expectFn: func(ms *mock_control.MockStreamer) {
				ms.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				ctx:   t.Context(),
				proc:  &mock_processor.MockConversations{},
				links: make(<-chan structures.EntityItem),
			},
			expectFn: func(ms *mock_control.MockStreamer) {
				ms.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "closed error",
			args: args{
				ctx:   t.Context(),
				proc:  &mock_processor.MockConversations{},
				links: make(<-chan structures.EntityItem),
			},
			expectFn: func(ms *mock_control.MockStreamer) {
				ms.EXPECT().Conversations(gomock.Any(), gomock.Any(), gomock.Any()).Return(transform.ErrClosed)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_control.NewMockStreamer(ctrl)
			tt.expectFn(ms)
			if err := conversationWorker(tt.args.ctx, ms, tt.args.proc, tt.args.links); (err != nil) != tt.wantErr {
				t.Errorf("conversationWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_workspaceWorker(t *testing.T) {
	type args struct {
		ctx context.Context
		// s      Streamer
		wsproc processor.WorkspaceInfo
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:    t.Context(),
				wsproc: &mock_processor.MockWorkspaceInfo{},
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "error",
			args: args{
				ctx:    t.Context(),
				wsproc: &mock_processor.MockWorkspaceInfo{},
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().WorkspaceInfo(gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			tt.expectFn(s)
			if err := workspaceWorker(tt.args.ctx, s, tt.args.wsproc); (err != nil) != tt.wantErr {
				t.Errorf("workspaceWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_searchMsgWorker(t *testing.T) {
	type args struct {
		ctx context.Context
		// s     Streamer
		ms    processor.MessageSearcher
		query string
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:   t.Context(),
				ms:    &mock_processor.MockMessageSearcher{},
				query: "query",
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "query").Return(nil)
			},
		},
		{
			name: "error",
			args: args{
				ctx:   t.Context(),
				ms:    &mock_processor.MockMessageSearcher{},
				query: "query",
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().SearchMessages(gomock.Any(), gomock.Any(), "query").Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			tt.expectFn(s)
			if err := searchMsgWorker(tt.args.ctx, s, tt.args.ms, tt.args.query); (err != nil) != tt.wantErr {
				t.Errorf("searchMsgWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_searchFileWorker(t *testing.T) {
	type args struct {
		ctx context.Context
		// s     Streamer
		sf    processor.FileSearcher
		query string
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer)
		wantErr  bool
	}{
		{
			name: "no errors",
			args: args{
				ctx:   t.Context(),
				sf:    &mock_processor.MockFileSearcher{},
				query: "query",
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "query").Return(nil)
			},
		},
		{
			name: "error",
			args: args{
				ctx:   t.Context(),
				sf:    &mock_processor.MockFileSearcher{},
				query: "query",
			},
			expectFn: func(s *mock_control.MockStreamer) {
				s.EXPECT().SearchFiles(gomock.Any(), gomock.Any(), "query").Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			tt.expectFn(s)
			if err := searchFileWorker(tt.args.ctx, s, tt.args.sf, tt.args.query); (err != nil) != tt.wantErr {
				t.Errorf("searchFileWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
