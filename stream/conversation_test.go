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

package stream

import (
	"context"
	"errors"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
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
	threadedMsg := []slack.Message{
		{Msg: slack.Msg{
			Timestamp:       "1577694990.000400",
			ThreadTimestamp: "1577694990.000400",
			LatestReply:     "1638784627.000300",
			ReplyCount:      3,
		}},
	}
	tests := []struct {
		name     string
		args     args
		skipFn   func(ctx context.Context, channelID, threadTS string, replyCount int) bool
		expectFn func(mp *mock_processor.MockConversations)
		want     int
		wantErr  bool
	}{
		{
			name: "empty messages slice",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      []slack.Message{},
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, []slack.Message{}).Times(1)
			},
		},
		{
			name: "empty message slice, processor error",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      []slack.Message{},
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, []slack.Message{}).Return(assert.AnError).Times(1)
			},
			wantErr: true,
		},
		{
			name: "non-empty messages slice",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)).Times(1)
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
		},
		{
			name: "non-empty messages slice,files processor error",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "non-empty messages slice, messages processor error",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport),
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)).Return(assert.AnError).Times(1)
				mp.EXPECT().Files(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			wantErr: true,
		},
		{
			name: "skip complete thread",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request),
				channel: TestChannel,
				isLast:  true,
				mm:      threadedMsg,
			},
			skipFn: func(_ context.Context, _, _ string, _ int) bool { return true },
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, threadedMsg).Times(1)
			},
			want: 0,
		},
		{
			name: "do not skip incomplete thread",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request, 1),
				channel: TestChannel,
				isLast:  true,
				mm:      threadedMsg,
			},
			skipFn: func(_ context.Context, _, _ string, _ int) bool { return false },
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 1, true, threadedMsg).Times(1)
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mp := mock_processor.NewMockConversations(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mp)
			}
			got, err := (&Stream{skipThread: tt.skipFn}).procChanMsg(tt.args.ctx, mp, tt.args.threadC, tt.args.channel, tt.args.isLast, tt.args.mm)
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

func stuffProcWithFiles(mp *mock_processor.MockConversations, ch *slack.Channel, mm []slack.Message) {
	for _, m := range mm {
		if len(m.Files) > 0 {
			mp.EXPECT().Files(gomock.Any(), ch, m, m.Files).Times(1)
		}
	}
}

func Test_procThreadMsg(t *testing.T) {
	testMessages := fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)
	type args struct {
		ctx context.Context
		// proc       processor.Conversations // supplied by test
		channel    *slack.Channel
		threadTS   string
		threadOnly bool
		isLast     bool
		msgs       []slack.Message
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mp *mock_processor.MockConversations)
		wantErr  bool
	}{
		{
			"empty messages slice",
			args{
				t.Context(),
				TestChannel,
				"123456.789",
				false,
				true,
				[]slack.Message{},
			},
			nil,
			false,
		},
		{
			"one message",
			args{
				t.Context(),
				TestChannel,
				"123456.789",
				false,
				true,
				testMessages[0:1],
			},
			func(mp *mock_processor.MockConversations) {
				mp.EXPECT().ThreadMessages(gomock.Any(), TestChannel.ID, testMessages[0], false, true, testMessages[0:1]).Times(1)
			},
			false,
		},
		{
			"all test messages",
			args{
				t.Context(),
				TestChannel,
				"123456.789",
				false,
				false,
				testMessages,
			},
			func(mp *mock_processor.MockConversations) {
				stuffProcWithFiles(mp, TestChannel, testMessages)
				mp.EXPECT().ThreadMessages(gomock.Any(), TestChannel.ID, testMessages[0], false, false, testMessages).Times(1)
			},
			false,
		},
		{
			"all test messages, files processor error",
			args{
				t.Context(),
				TestChannel,
				"123456.789",
				false,
				false,
				testMessages,
			},
			func(mp *mock_processor.MockConversations) {
				for _, m := range testMessages[1:] {
					if len(m.Files) > 0 {
						mp.EXPECT().Files(gomock.Any(), TestChannel, m, m.Files).Return(assert.AnError).Times(1)
						break
					}
				}
			},
			true,
		},
		{
			"all test messages, thread messages processor error",
			args{
				t.Context(),
				TestChannel,
				"123456.789",
				false,
				false,
				testMessages,
			},
			func(mp *mock_processor.MockConversations) {
				stuffProcWithFiles(mp, TestChannel, testMessages)
				mp.EXPECT().ThreadMessages(gomock.Any(), TestChannel.ID, testMessages[0], false, false, testMessages).Return(assert.AnError).Times(1)
			},
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
			if err := procThreadMsg(tt.args.ctx, mp, tt.args.channel, tt.args.threadTS, tt.args.threadOnly, tt.args.isLast, tt.args.msgs); (err != nil) != tt.wantErr {
				t.Errorf("procThreadMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_procFiles(t *testing.T) {
	testMessages := fixtures.Load[[]slack.Message](fixtures.TestChannelEveryoneMessagesNativeExport)
	type args struct {
		ctx context.Context
		// proc    processor.Filer // supplied by test
		channel *slack.Channel
		msgs    []slack.Message
	}
	tests := []struct {
		name    string
		args    args
		expect  func(mp *mock_processor.MockConversations)
		wantErr bool
	}{
		{
			"empty messages slice",
			args{
				t.Context(),
				TestChannel,
				[]slack.Message{},
			},
			nil,
			false,
		},
		{
			"all ok",
			args{
				t.Context(),
				TestChannel,
				testMessages,
			},
			func(mp *mock_processor.MockConversations) {
				stuffProcWithFiles(mp, TestChannel, testMessages)
			},
			false,
		},
		{
			"files processor error",
			args{
				t.Context(),
				TestChannel,
				testMessages,
			},
			func(mp *mock_processor.MockConversations) {
				for _, m := range testMessages {
					if len(m.Files) > 0 {
						mp.EXPECT().Files(gomock.Any(), TestChannel, m, m.Files).Return(assert.AnError).Times(1)
						break
					}
				}
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mp := mock_processor.NewMockConversations(ctrl)
			if tt.expect != nil {
				tt.expect(mp)
			}
			if err := procFiles(tt.args.ctx, mp, tt.args.channel, tt.args.msgs...); (err != nil) != tt.wantErr {
				t.Errorf("procFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_isNonCriticalErr(t *testing.T) {
	type args struct {
		e error
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		wantOK  bool
	}{
		{
			name:    "unknown error",
			args:    args{errors.New("foo")},
			wantErr: nil,
			wantOK:  false,
		},
		{
			name:    "channel not found",
			args:    args{slack.SlackErrorResponse{Err: errChanNotFound.Error()}},
			wantErr: errChanNotFound,
			wantOK:  true,
		},
		{
			name:    "not in channel",
			args:    args{slack.SlackErrorResponse{Err: errNotInChannel.Error()}},
			wantErr: errNotInChannel,
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, ok := isNonCriticalErr(tt.args.e)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("isNonCriticalErr() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Fatalf("isNonCriticalErr() ok = %t, wantOK = %t", ok, tt.wantOK)
			}
		})
	}
}
