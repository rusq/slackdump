package stream

import (
	"context"
	"testing"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var TestChannel = &slack.Channel{
	GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{
			ID: "C12345678",
		},
	},
}

func Test_procChanMsg(t *testing.T) {
	type args struct {
		ctx context.Context
		// proc    processor.Conversations // supplied by test
		threadC chan<- request
		channel *slack.Channel
		isLast  bool
		mm      []slack.Message
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mp *mock_processor.MockConversations)
		want     int
		wantErr  bool
	}{
		{
			"empty messages slice",
			args{
				context.Background(),
				make(chan request),
				TestChannel,
				true,
				[]slack.Message{},
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, []slack.Message{}).Times(1)
			},
			0,
			false,
		},
		{
			"empty message slice, processor error",
			args{
				context.Background(),
				make(chan request),
				TestChannel,
				true,
				[]slack.Message{},
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, []slack.Message{}).Return(assert.AnError).Times(1)
			},
			0,
			true,
		},
		{
			"non-empty messages slice",
			args{
				context.Background(),
				make(chan request),
				TestChannel,
				true,
				fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)).Times(1)
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			0,
			false,
		},
		{
			"non-empty messages slice,files processor error",
			args{
				context.Background(),
				make(chan request),
				TestChannel,
				true,
				fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			0,
			true,
		},
		{
			"non-empty messages slice, messages processor error",
			args{
				context.Background(),
				make(chan request),
				TestChannel,
				true,
				fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)).Return(assert.AnError).Times(1)
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mp := mock_processor.NewMockConversations(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mp)
			}
			got, err := procChanMsg(tt.args.ctx, mp, tt.args.threadC, tt.args.channel, tt.args.isLast, tt.args.mm)
			if (err != nil) != tt.wantErr {
				t.Errorf("procChanMsg() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("procChanMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}
