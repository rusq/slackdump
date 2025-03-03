package control

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/internal/chunk/dirproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	testUser1 = slack.User{
		ID:      "U12345678",
		TeamID:  "T11111111",
		Name:    "alice",
		Deleted: false,
	}
	testUser2 = slack.User{
		ID:      "U87654321",
		TeamID:  "T11111111",
		Name:    "bob",
		Deleted: false,
	}
)

func Test_userCollector_Users(t *testing.T) {
	testCtx := context.Background()
	type fields struct {
		ctx   context.Context
		users []slack.User
		ts    TransformStarter
	}
	type args struct {
		ctx   context.Context
		users []slack.User
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantErr   bool
		wantState *userCollector
	}{
		{
			name: "no users",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{},
			},
			args: args{
				ctx:   context.Background(),
				users: []slack.User{},
			},
			wantErr: false,
			wantState: &userCollector{
				ctx:   testCtx,
				users: []slack.User{},
			},
		},
		{
			name: "some users",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{},
			},
			args: args{
				ctx:   context.Background(),
				users: []slack.User{testUser1, testUser2},
			},
			wantErr: false,
			wantState: &userCollector{
				ctx:   testCtx,
				users: []slack.User{testUser1, testUser2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &userCollector{
				ctx:   tt.fields.ctx,
				users: tt.fields.users,
				ts:    tt.fields.ts,
			}
			if err := u.Users(tt.args.ctx, tt.args.users); (err != nil) != tt.wantErr {
				t.Errorf("Users() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantState, u)
		})
	}
}

func Test_userCollector_Close(t *testing.T) {
	testCtx := context.Background()
	type fields struct {
		ctx   context.Context
		users []slack.User
		// ts    TransformStarter
	}
	tests := []struct {
		name     string
		fields   fields
		expectFn func(*mock_control.MockTransformStarter)
		wantErr  bool
	}{
		{
			name: "no users is an error",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{},
			},
			wantErr: true,
		},
		{
			name: "transformer error",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{testUser1, testUser2},
			},
			expectFn: func(mts *mock_control.MockTransformStarter) {
				mts.EXPECT().StartWithUsers(gomock.Any(), []slack.User{testUser1, testUser2}).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "transformer success",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{testUser1, testUser2},
			},
			expectFn: func(mts *mock_control.MockTransformStarter) {
				mts.EXPECT().StartWithUsers(gomock.Any(), []slack.User{testUser1, testUser2}).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mts := mock_control.NewMockTransformStarter(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mts)
			}
			u := &userCollector{
				ctx:   tt.fields.ctx,
				users: tt.fields.users,
				ts:    mts,
			}
			if err := u.Close(); (err != nil) != tt.wantErr {
				t.Errorf("userCollector.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_conversationTransformer_mbeTransform(t *testing.T) {
	testCtx := context.Background()
	type fields struct {
		ctx context.Context
		// tf  dirproc.Transformer
		// rc  ReferenceChecker
	}
	type args struct {
		ctx        context.Context
		channelID  string
		threadID   string
		threadOnly bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_control.MockReferenceChecker, *mock_control.MockExportTransformer)
		wantErr  bool
	}{
		{
			name: "finalised",
			fields: fields{
				ctx: testCtx,
			},
			args: args{
				ctx:        testCtx,
				channelID:  "C12345678",
				threadID:   "",
				threadOnly: false,
			},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsFinalised(gomock.Any(), "C12345678").Return(true, nil)
				mes.EXPECT().Transform(gomock.Any(), chunk.FileID("C12345678")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "not finalised",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsFinalised(gomock.Any(), "C12345678").Return(false, nil)
			},
			wantErr: false,
		},
		{
			name:   "error checking finalised",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsFinalised(gomock.Any(), "C12345678").Return(false, assert.AnError)
			},
			wantErr: true,
		},
		{
			name:   "error transforming",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsFinalised(gomock.Any(), "C12345678").Return(true, nil)
				mes.EXPECT().Transform(gomock.Any(), chunk.FileID("C12345678")).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:   "thread",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678", threadID: "1234.5678", threadOnly: true},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsFinalised(gomock.Any(), "C12345678").Return(true, nil)
				mes.EXPECT().Transform(gomock.Any(), chunk.ToFileID("C12345678", "1234.5678", true)).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mrc := mock_control.NewMockReferenceChecker(ctrl)
			mes := mock_control.NewMockExportTransformer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mrc, mes)
			}
			ct := &conversationTransformer{
				ctx: tt.fields.ctx,
				tf:  mes,
				rc:  mrc,
			}
			if err := ct.mbeTransform(tt.args.ctx, tt.args.channelID, tt.args.threadID, tt.args.threadOnly); (err != nil) != tt.wantErr {
				t.Errorf("conversationTransformer.mbeTransform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_conversationTransformer_ThreadMessages(t *testing.T) {
	type fields struct {
		ctx      context.Context
		tf       dirproc.Transformer
		expectFn func(*mock_control.MockReferenceChecker, *mock_control.MockExportTransformer)
		rc       ReferenceChecker
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
		expectFn func(*mock_control.MockReferenceChecker, *mock_control.MockExportTransformer)
		wantErr  bool
	}{
		{
			name: "not last",
			fields: fields{
				ctx: context.Background(),
			},
			args: args{
				ctx:        context.Background(),
				channelID:  "C12345678",
				parent:     slack.Message{},
				threadOnly: false,
				isLast:     false,
				tm:         []slack.Message{},
			},
			wantErr: false,
		},
		{
			name: "last, no error",
			fields: fields{
				ctx: context.Background(),
			},
			args: args{
				ctx:        context.Background(),
				channelID:  "C12345678",
				parent:     slack.Message{},
				threadOnly: false,
				isLast:     true,
				tm:         []slack.Message{},
			},
			expectFn: ctMbeTransformSuccess,
			wantErr:  false,
		},
		{
			name: "last, error",
			fields: fields{
				ctx: context.Background(),
			},
			args: args{
				ctx:        context.Background(),
				channelID:  "C12345678",
				parent:     slack.Message{},
				threadOnly: false,
				isLast:     true,
				tm:         []slack.Message{},
			},
			expectFn: ctMbeTransformError,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mrc := mock_control.NewMockReferenceChecker(ctrl)
			mes := mock_control.NewMockExportTransformer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mrc, mes)
			}
			ct := &conversationTransformer{
				ctx: tt.fields.ctx,
				tf:  mes,
				rc:  mrc,
			}
			if err := ct.ThreadMessages(tt.args.ctx, tt.args.channelID, tt.args.parent, tt.args.threadOnly, tt.args.isLast, tt.args.tm); (err != nil) != tt.wantErr {
				t.Errorf("conversationTransformer.ThreadMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func ctMbeTransformError(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
	mrc.EXPECT().IsFinalised(gomock.Any(), gomock.Any()).Return(true, nil)
	mes.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(assert.AnError)
}

func ctMbeTransformSuccess(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
	mrc.EXPECT().IsFinalised(gomock.Any(), gomock.Any()).Return(true, nil)
	mes.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(nil)
}

func Test_conversationTransformer_Messages(t *testing.T) {
	testCtx := context.Background()
	type fields struct {
		ctx context.Context
		// tf  dirproc.Transformer
		// rc  ReferenceChecker
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
		expectFn func(*mock_control.MockReferenceChecker, *mock_control.MockExportTransformer)
		wantErr  bool
	}{
		{
			name: "not last",
			fields: fields{
				ctx: testCtx,
			},
			args: args{
				ctx:        testCtx,
				channelID:  "C12345678",
				numThreads: 0,
				isLast:     false,
				mm:         []slack.Message{},
			},
			wantErr: false,
		},
		{
			name: "last, no error",
			fields: fields{
				ctx: testCtx,
			},
			args: args{
				ctx:        testCtx,
				channelID:  "C12345678",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: ctMbeTransformSuccess,
			wantErr:  false,
		},
		{
			name: "last, no error",
			fields: fields{
				ctx: testCtx,
			},
			args: args{
				ctx:        testCtx,
				channelID:  "C12345678",
				numThreads: 0,
				isLast:     true,
				mm:         []slack.Message{},
			},
			expectFn: ctMbeTransformError,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mrc := mock_control.NewMockReferenceChecker(ctrl)
			mes := mock_control.NewMockExportTransformer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mrc, mes)
			}
			ct := &conversationTransformer{
				ctx: tt.fields.ctx,
				tf:  mes,
				rc:  mrc,
			}
			if err := ct.Messages(tt.args.ctx, tt.args.channelID, tt.args.numThreads, tt.args.isLast, tt.args.mm); (err != nil) != tt.wantErr {
				t.Errorf("conversationTransformer.Messages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chanFilter_Channels(t *testing.T) {
	type fields struct {
		links      chan<- structures.EntityItem
		list       *structures.EntityList
		memberOnly bool
		idx        map[string]*structures.EntityItem
	}
	type args struct {
		ctx context.Context
		ch  []slack.Channel
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
			c := &chanFilter{
				links:      tt.fields.links,
				list:       tt.fields.list,
				memberOnly: tt.fields.memberOnly,
				idx:        tt.fields.idx,
			}
			if err := c.Channels(tt.args.ctx, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("chanFilter.Channels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_combinedChannels_Channels(t *testing.T) {
	type fields struct {
		output    chan<- structures.EntityItem
		processed map[string]struct{}
	}
	type args struct {
		ctx context.Context
		ch  []slack.Channel
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
			c := &combinedChannels{
				output:    tt.fields.output,
				processed: tt.fields.processed,
			}
			if err := c.Channels(tt.args.ctx, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("combinedChannels.Channels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
