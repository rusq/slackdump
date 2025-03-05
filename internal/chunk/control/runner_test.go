package control

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/internal/chunk/control/mock_control"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/rusq/slackdump/v3/processor"
)

func Test_apiGenerator_Generate(t *testing.T) {
	type fields struct {
		// s          Streamer
		// p          processor.Channels
		memberOnly bool
		chTypes    []string
	}
	type args struct {
		ctx context.Context
		// errC chan<- error
		list *structures.EntityList
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_control.MockStreamer, *mock_processor.MockChannels)
		want     []structures.EntityItem
		wantErr  bool
	}{
		{
			name: "returns channels from the fake 'API'",
			fields: fields{
				memberOnly: false,
				chTypes:    []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				{Id: "C22222222", Include: true},
			},
			wantErr: false,
		},
		{
			name: "excludes excluded channels",
			fields: fields{
				memberOnly: false,
				chTypes:    []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(structures.EntityItem{Id: "C22222222", Include: false}),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				// second channel gets filtered out.
			},
			wantErr: false,
		},
		{
			name: "sets channel types if none are provided",
			fields: fields{
				memberOnly: false,
				chTypes:    nil,
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: slackdump.AllChanTypes}).Return(nil)
			},
			want:    []structures.EntityItem{},
			wantErr: false,
		},
		{
			name: "skips non-member channels",
			fields: fields{
				memberOnly: true,
				chTypes:    []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
			},
			wantErr: false,
		},
		{
			name: "handles error",
			fields: fields{
				memberOnly: false,
				chTypes:    []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					Return(assert.AnError)
			},
			want:    []structures.EntityItem{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			p := mock_processor.NewMockChannels(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(s, p)
			}
			g := &apiGenerator{
				s:          s,
				p:          p,
				memberOnly: tt.fields.memberOnly,
				chTypes:    tt.fields.chTypes,
			}

			errC := make(chan error, 1)
			listC := g.Generate(tt.args.ctx, errC, tt.args.list)
			collected := collectItems(listC)
			got := collected()
			close(errC)

			assert.Equal(t, tt.want, got)
			if err := <-errC; (err != nil) != tt.wantErr {
				t.Errorf("apiGenerator.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_combinedGenerator_Generate(t *testing.T) {
	date1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	type fields struct {
		// s       Streamer
		// p       processor.Channels
		chTypes []string
	}
	type args struct {
		ctx context.Context
		// errC chan<- error
		list *structures.EntityList
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expectFn func(*mock_control.MockStreamer, *mock_processor.MockChannels)
		want     []structures.EntityItem
		wantErr  bool
	}{
		{
			name: "returns channels from the list",
			fields: fields{
				chTypes: []string{"public_channel"},
			},
			args: args{
				ctx: context.Background(),
				list: structures.NewEntityListFromItems(
					structures.EntityItem{
						Id:      "C11111111",
						Include: true,
						Oldest:  date1,
						Latest:  date2,
					},
					structures.EntityItem{
						Id:      "C22222222",
						Include: true,
					},
				),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true, Oldest: date1, Latest: date2},
				{Id: "C22222222", Include: true},
			},
			wantErr: false,
		},
		{
			name: "returns channels from the list and the API",
			fields: fields{
				chTypes: nil,
			},
			args: args{
				ctx: context.Background(),
				list: structures.NewEntityListFromItems(
					structures.EntityItem{Id: "C11111111", Include: true, Oldest: date1, Latest: date2},
				),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: slackdump.AllChanTypes}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember, testGroupChanNonMember, testPrivChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember, testGroupChanNonMember, testPrivChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true, Oldest: date1, Latest: date2},
				{Id: "C22222222", Include: true},
				{Id: "G44444444", Include: true},
				{Id: "D33333333", Include: true},
			},
			wantErr: false,
		},
		{
			name: "returns channels from the API, if no list",
			fields: fields{
				chTypes: []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true},
				{Id: "C22222222", Include: true},
			},
			wantErr: false,
		},
		{
			name: "does not reprocess channels from the list",
			fields: fields{
				chTypes: []string{"public_channel"},
			},
			args: args{
				ctx: context.Background(),
				list: structures.NewEntityListFromItems(
					structures.EntityItem{Id: "C11111111", Include: true, Oldest: date1, Latest: date2},
				),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					DoAndReturn(
						func(ctx context.Context, proc processor.Channels, p *slack.GetConversationsParameters) error {
							proc.Channels(context.Background(), []slack.Channel{testPubChanMember, testPubChanNonMember})
							return nil
						})
				p.EXPECT().Channels(gomock.Any(), []slack.Channel{testPubChanMember, testPubChanNonMember}).Return(nil)
			},
			want: []structures.EntityItem{
				{Id: "C11111111", Include: true, Oldest: date1, Latest: date2}, // this one is from the list, it has the dates.
				{Id: "C22222222", Include: true},
			},
			wantErr: false,
		},
		{
			name: "handles error",
			fields: fields{
				chTypes: []string{"public_channel"},
			},
			args: args{
				ctx:  context.Background(),
				list: structures.NewEntityListFromItems(),
			},
			expectFn: func(s *mock_control.MockStreamer, p *mock_processor.MockChannels) {
				s.EXPECT().
					ListChannels(gomock.Any(), gomock.Any(), &slack.GetConversationsParameters{Types: []string{"public_channel"}}).
					Return(assert.AnError)
			},
			want:    []structures.EntityItem{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			p := mock_processor.NewMockChannels(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(s, p)
			}
			g := &combinedGenerator{
				s:       s,
				p:       p,
				chTypes: tt.fields.chTypes,
			}
			errC := make(chan error, 1)
			listC := g.Generate(tt.args.ctx, errC, tt.args.list)
			collected := collectItems(listC)
			got := collected()
			close(errC)

			assert.Equal(t, tt.want, got)
			if err := <-errC; (err != nil) != tt.wantErr {
				t.Errorf("apiGenerator.Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type fakeCloser struct {
	err error
}

func (f *fakeCloser) Close() error {
	return f.err
}

func Test_tryClose(t *testing.T) {
	type args struct {
		// errC chan<- error
		a any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no error, closer",
			args: args{
				a: &fakeCloser{},
			},
			wantErr: false,
		},
		{
			name: "error, closer",
			args: args{
				a: &fakeCloser{err: assert.AnError},
			},
			wantErr: true,
		},
		{
			name: "no error, not a closer",
			args: args{
				a: struct{}{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errC := make(chan error, 1)
			tryClose(errC, tt.args.a)
			close(errC)
			err := <-errC
			if (err != nil) != tt.wantErr {
				t.Errorf("tryClose() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newGenerator(t *testing.T) {
	type args struct {
		s     Streamer
		p     superprocessor
		flags Flags
		list  *structures.EntityList
	}
	tests := []struct {
		name string
		args args
		want generator
	}{
		{
			name: "refresh",
			args: args{
				s:     nil,
				flags: Flags{Refresh: true},
				list:  nil,
			},
			want: &combinedGenerator{},
		},
		{
			name: "inclusive",
			args: args{
				s:     nil,
				flags: Flags{},
				list:  structures.NewEntityListFromItems(structures.EntityItem{Id: "C11111111", Include: true}),
			},
			want: &listGen{},
		},
		{
			name: "exclusive",
			args: args{
				s:     nil,
				flags: Flags{},
				list:  structures.NewEntityListFromItems(structures.EntityItem{Id: "C11111111", Include: false}),
			},
			want: &apiGenerator{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newGenerator(tt.args.s, tt.args.p, tt.args.flags, tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runWorkers(t *testing.T) {
	type superMockProcessor struct {
		*mock_processor.MockConversations
		*mock_processor.MockUsers
		*mock_processor.MockChannels
		*mock_processor.MockWorkspaceInfo
	}
	testList := structures.NewEntityListFromItems(
		structures.EntityItem{Id: "C11111111", Include: true},
	)
	type args struct {
		ctx context.Context
		// s     Streamer
		list  *structures.EntityList
		p     superprocessor
		flags Flags
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(*mock_control.MockStreamer, *superMockProcessor)
		wantErr  bool
	}{
		{
			name: "one channel",
			args: args{
				ctx:   context.Background(),
				list:  testList,
				flags: Flags{},
			},
			expectFn: func(s *mock_control.MockStreamer, m *superMockProcessor) {
				s.EXPECT().
					WorkspaceInfo(gomock.Any(), m.MockWorkspaceInfo).
					Return(nil)
				s.EXPECT().
					Conversations(gomock.Any(), m.MockConversations, gomock.Any()).
					Return(nil)
				s.EXPECT().
					Users(gomock.Any(), m.MockUsers, gomock.Any()).
					Return(nil)
				m.MockConversations.EXPECT().Close().Return(nil)
			},
			wantErr: false,
		},
		{
			name: "conversations error",
			args: args{
				ctx:   context.Background(),
				list:  testList,
				flags: Flags{},
			},
			expectFn: func(s *mock_control.MockStreamer, m *superMockProcessor) {
				s.EXPECT().
					WorkspaceInfo(gomock.Any(), m.MockWorkspaceInfo).
					Return(nil)
				s.EXPECT().
					Conversations(gomock.Any(), m.MockConversations, gomock.Any()).
					Return(assert.AnError)
				s.EXPECT().
					Users(gomock.Any(), m.MockUsers, gomock.Any()).
					Return(nil)
				m.MockConversations.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "workspace info error",
			args: args{
				ctx:   context.Background(),
				list:  testList,
				flags: Flags{},
			},
			expectFn: func(s *mock_control.MockStreamer, m *superMockProcessor) {
				s.EXPECT().
					WorkspaceInfo(gomock.Any(), m.MockWorkspaceInfo).
					Return(assert.AnError)
				s.EXPECT().
					Conversations(gomock.Any(), m.MockConversations, gomock.Any()).
					Return(nil)
				s.EXPECT().
					Users(gomock.Any(), m.MockUsers, gomock.Any()).
					Return(nil)
				m.MockConversations.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "users error",
			args: args{
				ctx:   context.Background(),
				list:  testList,
				flags: Flags{},
			},
			expectFn: func(s *mock_control.MockStreamer, m *superMockProcessor) {
				s.EXPECT().
					WorkspaceInfo(gomock.Any(), m.MockWorkspaceInfo).
					Return(nil)
				s.EXPECT().
					Conversations(gomock.Any(), m.MockConversations, gomock.Any()).
					Return(nil)
				s.EXPECT().
					Users(gomock.Any(), m.MockUsers, gomock.Any()).
					Return(assert.AnError)
				m.MockConversations.EXPECT().Close().Return(nil)
			},
			wantErr: true,
		},
		{
			name: "close error",
			args: args{
				ctx:   context.Background(),
				list:  testList,
				flags: Flags{},
			},
			expectFn: func(s *mock_control.MockStreamer, m *superMockProcessor) {
				s.EXPECT().
					WorkspaceInfo(gomock.Any(), m.MockWorkspaceInfo).
					Return(nil)
				s.EXPECT().
					Conversations(gomock.Any(), m.MockConversations, gomock.Any()).
					Return(nil)
				s.EXPECT().
					Users(gomock.Any(), m.MockUsers, gomock.Any()).
					Return(nil)
				m.MockConversations.EXPECT().Close().Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := mock_control.NewMockStreamer(ctrl)
			m := &superMockProcessor{
				MockConversations: mock_processor.NewMockConversations(ctrl),
				MockUsers:         mock_processor.NewMockUsers(ctrl),
				MockChannels:      mock_processor.NewMockChannels(ctrl),
				MockWorkspaceInfo: mock_processor.NewMockWorkspaceInfo(ctrl),
			}
			if tt.expectFn != nil {
				tt.expectFn(s, m)
			}
			p := superprocessor{
				Conversations: m.MockConversations,
				Users:         m.MockUsers,
				Channels:      m.MockChannels,
				WorkspaceInfo: m.MockWorkspaceInfo,
			}
			if err := runWorkers(tt.args.ctx, s, tt.args.list, p, tt.args.flags); (err != nil) != tt.wantErr {
				t.Errorf("runWorkers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
