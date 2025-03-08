package control

import (
	"context"
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
		{
			name: "cancelled context",
			fields: fields{
				ctx:   testCtx,
				users: []slack.User{testUser1, testUser2},
			},
			expectFn: func(mts *mock_control.MockTransformStarter) {
				mts.EXPECT().StartWithUsers(gomock.Any(), gomock.Any()).Return(context.Canceled)
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
				mrc.EXPECT().IsComplete(gomock.Any(), "C12345678").Return(true, nil)
				mes.EXPECT().Transform(gomock.Any(), chunk.FileID("C12345678")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "not finalised",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsComplete(gomock.Any(), "C12345678").Return(false, nil)
			},
			wantErr: false,
		},
		{
			name:   "error checking finalised",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsComplete(gomock.Any(), "C12345678").Return(false, assert.AnError)
			},
			wantErr: true,
		},
		{
			name:   "error transforming",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678"},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsComplete(gomock.Any(), "C12345678").Return(true, nil)
				mes.EXPECT().Transform(gomock.Any(), chunk.FileID("C12345678")).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name:   "thread",
			fields: fields{ctx: testCtx},
			args:   args{ctx: testCtx, channelID: "C12345678", threadID: "1234.5678", threadOnly: true},
			expectFn: func(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
				mrc.EXPECT().IsComplete(gomock.Any(), "C12345678").Return(true, nil)
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
		tf       chunk.Transformer
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
	mrc.EXPECT().IsComplete(gomock.Any(), gomock.Any()).Return(true, nil)
	mes.EXPECT().Transform(gomock.Any(), gomock.Any()).Return(assert.AnError)
}

func ctMbeTransformSuccess(mrc *mock_control.MockReferenceChecker, mes *mock_control.MockExportTransformer) {
	mrc.EXPECT().IsComplete(gomock.Any(), gomock.Any()).Return(true, nil)
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

var (
	testPubChanMember = slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "C11111111",
			},
			Name: "public",
		},
		IsMember: true,
	}

	testPubChanNonMember = slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "C22222222",
			},
			Name: "public2",
		},
		IsMember: false,
	}

	testPrivChanNonMember = slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "D33333333",
			},
			Name: "private",
		},
		IsMember: false,
	}

	testGroupChanNonMember = slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "G44444444",
			},
			Name: "group",
		},
		IsMember: false,
	}
)

func Test_chanFilter_Channels(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	type fields struct {
		// links      chan<- structures.EntityItem
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
		want    []structures.EntityItem
		wantErr bool
	}{
		{
			name: "test public channel member only",
			fields: fields{
				memberOnly: true,
				idx:        make(map[string]*structures.EntityItem),
			},
			args: args{
				ctx: context.Background(),
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				{Id: "D33333333", Include: true},
				{Id: "G44444444", Include: true},
			},
			wantErr: false,
		},
		{
			name: "includes all channels if memberOnly is false",
			fields: fields{
				memberOnly: false,
				idx:        make(map[string]*structures.EntityItem),
			},
			args: args{
				ctx: context.Background(),
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				{Id: "C22222222", Include: true},
				{Id: "D33333333", Include: true},
				{Id: "G44444444", Include: true},
			},
			wantErr: false,
		},
		{
			name: "skips excluded channels",
			fields: fields{
				memberOnly: false,
				idx:        must(structures.NewEntityList([]string{"^C11111111", "^G44444444"})).Index(),
			},
			args: args{
				ctx: context.Background(),
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want: []structures.EntityItem{
				{Id: "C22222222", Include: true},
				{Id: "D33333333", Include: true},
			},
		},
		{
			name: "cancelled context",
			fields: fields{
				memberOnly: false,
				idx:        make(map[string]*structures.EntityItem),
			},
			args: args{
				ctx: cancelledCtx,
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
				},
			},
			want:    []structures.EntityItem{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linksC := make(chan structures.EntityItem)
			c := &chanFilter{
				links:      linksC,
				memberOnly: tt.fields.memberOnly,
				idx:        tt.fields.idx,
			}

			collected := collectItems(linksC)

			if err := c.Channels(tt.args.ctx, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("chanFilter.Channels() error = %v, wantErr %v", err, tt.wantErr)
			}
			close(linksC)

			got := collected()
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_combinedChannels_Channels(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	type fields struct {
		// output    chan<- structures.EntityItem
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
		want    []structures.EntityItem
		wantErr bool
	}{
		{
			name: "no processed channels",
			fields: fields{
				processed: make(map[string]struct{}),
			},
			args: args{
				ctx: context.Background(),
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				{Id: "C22222222", Include: true},
				{Id: "D33333333", Include: true},
				{Id: "G44444444", Include: true},
			},
			wantErr: false,
		},
		{
			name: "skips processed channels",
			fields: fields{
				processed: map[string]struct{}{
					"C11111111": {},
					"D33333333": {},
				},
			},
			args: args{
				ctx: context.Background(),
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want: []structures.EntityItem{
				{Id: "C22222222", Include: true},
				{Id: "G44444444", Include: true},
			},
			wantErr: false,
		},
		{
			name: "cancelled context",
			fields: fields{
				processed: make(map[string]struct{}),
			},
			args: args{
				ctx: cancelledCtx,
				ch: []slack.Channel{
					testPubChanMember,
					testPubChanNonMember,
					testPrivChanNonMember,
					testGroupChanNonMember,
				},
			},
			want:    []structures.EntityItem{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputC := make(chan structures.EntityItem)
			c := &combinedChannels{
				output:    outputC,
				processed: tt.fields.processed,
			}

			collected := collectItems(outputC)
			if err := c.Channels(tt.args.ctx, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("combinedChannels.Channels() error = %v, wantErr %v", err, tt.wantErr)
			}
			close(outputC)

			got := collected()
			assert.Equal(t, tt.want, got)
		})
	}
}

// collectItems starts a goroutine to collect the items.  When the return
// function is called, it waits for the goroutine to finish.  It returns the
// collected items.
func collectItems[T any](c <-chan T) func() []T {
	done := make(chan struct{})
	items := make([]T, 0)
	go func() {
		defer close(done)
		for item := range c {
			items = append(items, item)
		}
	}()
	return func() []T {
		<-done
		return items
	}
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func Test_errEmitter(t *testing.T) {
	type args struct {
		// errC  chan<- error
		sub   string
		stage Stage
	}
	tests := []struct {
		name string
		args args
		call error
		want error
	}{
		{
			name: "emits error",
			args: args{
				sub:   "test",
				stage: "unit",
			},
			call: assert.AnError,
			want: Error{
				Subroutine: "test",
				Stage:      "unit",
				Err:        assert.AnError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errC := make(chan error, 1)
			e := errEmitter(errC, tt.args.sub, tt.args.stage)
			e(tt.call)
			assert.Equal(t, tt.want, <-errC)
		})
	}
}

func Test_jointFileSearcher_Files(t *testing.T) {
	type args struct {
		ctx   context.Context
		ch    *slack.Channel
		msg   slack.Message
		files []slack.File
	}
	tests := []struct {
		name     string
		expectFn func(*mock_processor.MockFileSearcher, *mock_processor.MockFiler)
		args     args
		wantErr  bool
	}{
		{
			name: "no error",
			expectFn: func(mfs *mock_processor.MockFileSearcher, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			args:    args{},
			wantErr: false,
		},
		{
			name: "error",
			expectFn: func(mfs *mock_processor.MockFileSearcher, mf *mock_processor.MockFiler) {
				mf.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)
			},
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfs := mock_processor.NewMockFileSearcher(ctrl)
			mf := mock_processor.NewMockFiler(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mfs, mf)
			}
			j := &jointFileSearcher{
				FileSearcher: mfs,
				filer:        mf,
			}
			// ensure the interface is implemented, and the right method is called
			var ifs processor.FileSearcher = j
			if err := ifs.Files(tt.args.ctx, tt.args.ch, tt.args.msg, tt.args.files); (err != nil) != tt.wantErr {
				t.Errorf("jointFileSearcher.Files() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_jointFileSearcher_Close(t *testing.T) {
	tests := []struct {
		name     string
		expectFn func(*mock_processor.MockFileSearcher, *mock_processor.MockFiler)
		wantErr  bool
	}{
		{
			name: "no error",
			expectFn: func(mfs *mock_processor.MockFileSearcher, mf *mock_processor.MockFiler) {
				mfs.EXPECT().Close().Return(nil).Times(1)
				mf.EXPECT().Close().Return(nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "error",
			expectFn: func(mfs *mock_processor.MockFileSearcher, mf *mock_processor.MockFiler) {
				mfs.EXPECT().Close().Return(assert.AnError).Times(1)
				mf.EXPECT().Close().Return(assert.AnError).Times(1)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfs := mock_processor.NewMockFileSearcher(ctrl)
			mf := mock_processor.NewMockFiler(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mfs, mf)
			}
			j := &jointFileSearcher{
				FileSearcher: mfs,
				filer:        mf,
			}
			if err := j.Close(); (err != nil) != tt.wantErr {
				t.Errorf("jointFileSearcher.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
