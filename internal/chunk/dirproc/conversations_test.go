package dirproc

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/rusq/slackdump/v3/processor"
	"go.uber.org/mock/gomock"
)

func TestConversations_Messages(t *testing.T) {
	textCtx := context.Background()
	type fields struct {
		subproc     processor.Filer
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx        context.Context
		channelID  string
		numThreads int
		isLast     bool
		mm         []slack.Message
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mh *Mockdatahandler)
		wantErr  bool
	}{
		{
			name: "ok, not a last message",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 500,
				isLast:     false,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Add(500).Return(501)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 500, false, []slack.Message{}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "processor error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 500,
				isLast:     false,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Add(500).Return(501)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 500, false, []slack.Message{}).Return(errors.New("processor error"))
			},
			wantErr: true,
		},
		{
			name: "ok, last message, but refcount is > 0",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Add(0).Return(1)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 0, true, []slack.Message{}).Return(nil)
				mh.EXPECT().Dec().Return(0)
				mt.EXPECT().RefCount(chunk.ToFileID("channelID", "", false)).Return(1)
			},
			wantErr: false,
		},
		{
			name: "ok, last message, refcount is = 0",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Add(0).Return(2)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 0, true, []slack.Message{}).Return(nil)
				mh.EXPECT().Dec().Return(1)
				mt.EXPECT().RefCount(chunk.ToFileID("channelID", "", false)).Return(0)
				mt.EXPECT().Unregister(chunk.ToFileID("channelID", "", false)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "recorder error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(nil, errors.New("recorder error"))
			},
			wantErr: true,
		},
		{
			name: "empty message slice, not last",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 0,
				isLast:     false,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 0, false, []slack.Message{}).Return(nil)
				mh.EXPECT().Add(0).Return(0)
			},
			wantErr: false,
		},
		{
			name: "empty message slice, last",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().Messages(gomock.Any(), "channelID", 0, true, []slack.Message{}).Return(nil)
				mh.EXPECT().Add(0).Return(1)
				mh.EXPECT().Dec().Return(0)
				mt.EXPECT().RefCount(chunk.ToFileID("channelID", "", false)).Return(0)
				mt.EXPECT().Unregister(chunk.ToFileID("channelID", "", false)).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mt := NewMocktracker(ctrl)
			mh := NewMockdatahandler(ctrl)
			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			tt.expectFn(mt, mh)
			if err := cv.Messages(tt.args.ctx, tt.args.channelID, tt.args.numThreads, tt.args.isLast, tt.args.mm); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.Messages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversations_ThreadMessages(t *testing.T) {
	textCtx := context.Background()
	type fields struct {
		subproc     processor.Filer
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx        context.Context
		channelID  string
		parent     slack.Message
		threadOnly bool
		isLast     bool
		tm         []slack.Message
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mh *Mockdatahandler)
		wantErr  bool
	}{
		{
			name: "ok, not a last message",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				parent:     slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				threadOnly: false,
				isLast:     false,
				tm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().ThreadMessages(gomock.Any(), "channelID", gomock.Any(), false, false, []slack.Message{}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "processor error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				parent:     slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				threadOnly: false,
				isLast:     false,
				tm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().ThreadMessages(gomock.Any(), "channelID", gomock.Any(), false, false, []slack.Message{}).Return(errors.New("processor error"))
			},
			wantErr: true,
		},
		{
			name: "ok, last message, but refcount is > 0",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				parent:     slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				threadOnly: false,
				isLast:     true,
				tm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().ThreadMessages(gomock.Any(), "channelID", gomock.Any(), false, true, []slack.Message{}).Return(nil)
				mh.EXPECT().Dec().Return(1)
				mt.EXPECT().RefCount(chunk.ToFileID("channelID", "123", false)).Return(1)
			},
			wantErr: false,
		},
		{
			name: "ok, last message, refcount is = 0",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				parent:     slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				threadOnly: false,
				isLast:     true,
				tm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(mh, nil)
				mh.EXPECT().ThreadMessages(gomock.Any(), "channelID", gomock.Any(), false, true, []slack.Message{}).Return(nil)
				mh.EXPECT().Dec().Return(0)
				mt.EXPECT().RefCount(chunk.ToFileID("channelID", "123", false)).Return(0)
				mt.EXPECT().Unregister(chunk.ToFileID("channelID", "123", false)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "recorder error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:        textCtx,
				channelID:  "channelID",
				parent:     slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				threadOnly: false,
				isLast:     true,
				tm:         []slack.Message{},
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(gomock.Any()).Return(nil, errors.New("recorder error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mt := NewMocktracker(ctrl)
			mh := NewMockdatahandler(ctrl)
			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			tt.expectFn(mt, mh)
			if err := cv.ThreadMessages(tt.args.ctx, tt.args.channelID, tt.args.parent, tt.args.threadOnly, tt.args.isLast, tt.args.tm); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.ThreadMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversations_ChannelInfo(t *testing.T) {
	textCtx := context.Background()
	type fields struct {
		subproc     processor.Filer
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx      context.Context
		ci       *slack.Channel
		threadTS string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mh *Mockdatahandler)
		wantErr  bool
	}{
		{
			name: "ok",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:      textCtx,
				ci:       fixtures.DummyChannel("channelID"),
				threadTS: "123",
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(mh, nil)
				mh.EXPECT().ChannelInfo(gomock.Any(), fixtures.DummyChannel("channelID"), "123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "recorder error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:      textCtx,
				ci:       fixtures.DummyChannel("channelID"),
				threadTS: "123",
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(nil, errors.New("recorder error"))
			},
			wantErr: true,
		},
		{
			name: "processor error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:      textCtx,
				ci:       fixtures.DummyChannel("channelID"),
				threadTS: "123",
			},
			expectFn: func(mt *Mocktracker, mh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(mh, nil)
				mh.EXPECT().ChannelInfo(gomock.Any(), fixtures.DummyChannel("channelID"), "123").Return(errors.New("processor error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mt := NewMocktracker(ctrl)
			mh := NewMockdatahandler(ctrl)

			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			tt.expectFn(mt, mh)
			if err := cv.ChannelInfo(tt.args.ctx, tt.args.ci, tt.args.threadTS); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.ChannelInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversations_finalise(t *testing.T) {
	textCtx := context.Background()
	type fields struct {
		subproc     processor.Filer
		recordFiles bool
	}
	type args struct {
		ctx context.Context
		id  chunk.FileID
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mtf *MockTransformer)
		wantErr  bool
	}{
		{
			name: "ok (refcount 0)",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
			},
			args: args{
				ctx: textCtx,
				id:  chunk.FileID("fileID"),
			},
			expectFn: func(mt *Mocktracker, mtf *MockTransformer) {
				mt.EXPECT().RefCount(chunk.FileID("fileID")).Return(0)
				mt.EXPECT().Unregister(chunk.FileID("fileID")).Return(nil)
				mtf.EXPECT().Transform(gomock.Any(), chunk.FileID("fileID")).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "ok (refcount > 0)",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
			},
			args: args{
				ctx: textCtx,
				id:  chunk.FileID("fileID"),
			},
			expectFn: func(mt *Mocktracker, mtf *MockTransformer) {
				mt.EXPECT().RefCount(chunk.FileID("fileID")).Return(1)
			},
			wantErr: false,
		},
		{
			name: "unregister error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
			},
			args: args{
				ctx: textCtx,
				id:  chunk.FileID("fileID"),
			},
			expectFn: func(mt *Mocktracker, mtf *MockTransformer) {
				mt.EXPECT().RefCount(chunk.FileID("fileID")).Return(0)
				mt.EXPECT().Unregister(chunk.FileID("fileID")).Return(errors.New("unregister error"))
			},
			wantErr: true,
		},
		{
			name: "transform error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
			},
			args: args{
				ctx: textCtx,
				id:  chunk.FileID("fileID"),
			},
			expectFn: func(mt *Mocktracker, mtf *MockTransformer) {
				mt.EXPECT().RefCount(chunk.FileID("fileID")).Return(0)
				mt.EXPECT().Unregister(chunk.FileID("fileID")).Return(nil)
				mtf.EXPECT().Transform(gomock.Any(), chunk.FileID("fileID")).Return(errors.New("transform error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mt := NewMocktracker(ctrl)
			mtf := NewMockTransformer(ctrl)
			tt.expectFn(mt, mtf)
			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          mtf,
			}
			if err := cv.finalise(tt.args.ctx, tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.finalise() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversations_Files(t *testing.T) {
	testCtx := context.Background()
	type fields struct {
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx     context.Context
		channel *slack.Channel
		parent  slack.Message
		ff      []slack.File
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mdh *Mockdatahandler, mf *mock_processor.MockFiler)
		wantErr  bool
	}{
		{
			name: "ok, recordFiles is false",
			fields: fields{
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:     testCtx,
				channel: fixtures.DummyChannel("channelID"),
				parent:  slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				ff:      []slack.File{},
			},
			expectFn: func(mt *Mocktracker, mfh *Mockdatahandler, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "ok, recordFiles is true",
			fields: fields{
				recordFiles: true,
				tf:          nil,
			},
			args: args{
				ctx:     testCtx,
				channel: fixtures.DummyChannel("channelID"),
				parent:  slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				ff:      []slack.File{},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(nil)
				mdh.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(nil)
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", false)).Return(mdh, nil)
			},
			wantErr: false,
		},
		{
			name: "subprocessor files returns error",
			fields: fields{
				recordFiles: true,
				tf:          nil,
			},
			args: args{
				ctx:     testCtx,
				channel: fixtures.DummyChannel("channelID"),
				parent:  slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				ff:      []slack.File{},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(errors.New("subprocessor error"))
			},
			wantErr: true,
		},
		{
			name: "recorder returns error",
			fields: fields{
				recordFiles: true,
				tf:          nil,
			},
			args: args{
				ctx:     testCtx,
				channel: fixtures.DummyChannel("channelID"),
				parent:  slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				ff:      []slack.File{},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(nil)
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", false)).Return(nil, errors.New("recorder error"))
			},
			wantErr: true,
		},
		{
			name: "datahandler returns error",
			fields: fields{
				recordFiles: true,
				tf:          nil,
			},
			args: args{
				ctx:     testCtx,
				channel: fixtures.DummyChannel("channelID"),
				parent:  slack.Message{Msg: slack.Msg{Timestamp: "123"}},
				ff:      []slack.File{},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(nil)
				mdh.EXPECT().Files(gomock.Any(), fixtures.DummyChannel("channelID"), slack.Message{Msg: slack.Msg{Timestamp: "123"}}, []slack.File{}).Return(errors.New("datahandler error"))
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", false)).Return(mdh, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mt := NewMocktracker(ctrl)
			mfh := NewMockdatahandler(ctrl)
			mf := mock_processor.NewMockFiler(ctrl)
			tt.expectFn(mt, mfh, mf)
			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       mf,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			if err := cv.Files(tt.args.ctx, tt.args.channel, tt.args.parent, tt.args.ff); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.Files() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConversations_ChannelUsers(t *testing.T) {
	textCtx := context.Background()
	type fields struct {
		subproc     processor.Filer
		recordFiles bool
		tf          Transformer
	}
	type args struct {
		ctx       context.Context
		channelID string
		threadTS  string
		cu        []string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(mt *Mocktracker, mdh *Mockdatahandler)
		wantErr  bool
	}{
		{
			name: "ok",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:       textCtx,
				channelID: "channelID",
				threadTS:  "123",
				cu:        []string{"user1", "user2"},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(mdh, nil)
				mdh.EXPECT().ChannelUsers(gomock.Any(), "channelID", "123", []string{"user1", "user2"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "recorder error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:       textCtx,
				channelID: "channelID",
				threadTS:  "123",
				cu:        []string{"user1", "user2"},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(nil, errors.New("recorder error"))
			},
			wantErr: true,
		},
		{
			name: "processor error",
			fields: fields{
				subproc:     nil,
				recordFiles: false,
				tf:          nil,
			},
			args: args{
				ctx:       textCtx,
				channelID: "channelID",
				threadTS:  "123",
				cu:        []string{"user1", "user2"},
			},
			expectFn: func(mt *Mocktracker, mdh *Mockdatahandler) {
				mt.EXPECT().Recorder(chunk.ToFileID("channelID", "123", true)).Return(mdh, nil)
				mdh.EXPECT().ChannelUsers(gomock.Any(), "channelID", "123", []string{"user1", "user2"}).Return(errors.New("processor error"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mt := NewMocktracker(ctrl)
			mdh := NewMockdatahandler(ctrl)
			tt.expectFn(mt, mdh)
			cv := &Conversations{
				t:           mt,
				lg:          slog.Default(),
				filer:       tt.fields.subproc,
				recordFiles: tt.fields.recordFiles,
				tf:          tt.fields.tf,
			}
			if err := cv.ChannelUsers(tt.args.ctx, tt.args.channelID, tt.args.threadTS, tt.args.cu); (err != nil) != tt.wantErr {
				t.Errorf("Conversations.ChannelUsers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Conversations_Close(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mt := NewMocktracker(ctrl)
		cv := &Conversations{
			t: mt,
		}
		mt.EXPECT().CloseAll().Return(nil)
		if err := cv.Close(); err != nil {
			t.Errorf("Conversations.Close() error = %v, wantErr %v", err, false)
		}
	})
}

func TestNewConversation(t *testing.T) {
	cd := &chunk.Directory{}
	ctrl := gomock.NewController(t)
	filesSubproc := mock_processor.NewMockFiler(ctrl)
	tf := NewMockTransformer(ctrl)

	t.Run("ok", func(t *testing.T) {
		// Test with valid arguments
		c, err := NewConversation(cd, filesSubproc, tf)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if c == nil {
			t.Errorf("Expected Conversations, got nil")
		}
	})

	t.Run("subprocessor validation", func(t *testing.T) {
		// Test with nil subprocessor
		_, err := NewConversation(cd, nil, tf)
		if !errors.Is(err, errNilSubproc) {
			t.Errorf("Expected 'internal error: files subprocessor is nil', got %v", err)
		}
	})

	t.Run("transformer validation", func(t *testing.T) {
		// Test with nil transformer
		_, err := NewConversation(cd, filesSubproc, nil)
		if !errors.Is(err, errNilTransformer) {
			t.Errorf("Expected 'internal error: transformer is nil', got %v", err)
		}
	})
}
