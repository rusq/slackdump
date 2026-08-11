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
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/client/mock_client"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
)

var TestChannel = &slack.Channel{
	GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{
			ID: "C12345678",
		},
	},
}

func Test_produceItems(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	output := make(chan structures.EntityItem)
	done := make(chan struct{})
	go func() {
		defer close(done)
		produceItems(ctx, output, []structures.EntityItem{{Id: "C1"}})
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("produceItems remained blocked after cancellation")
	}
}

func Test_runConversationItems(t *testing.T) {
	consumerErr := errors.New("consumer failed")
	cancelObserved := make(chan struct{})
	releaseProducer := make(chan struct{})
	producerDone := make(chan struct{})
	producer := func(ctx context.Context, output chan<- structures.EntityItem, _ []structures.EntityItem) {
		defer close(output)
		defer close(producerDone)
		output <- structures.EntityItem{Id: "C1"}
		<-ctx.Done()
		close(cancelObserved)
		<-releaseProducer
	}
	consumer := func(_ context.Context, input <-chan structures.EntityItem) error {
		<-input
		return consumerErr
	}

	result := make(chan error, 1)
	go func() {
		result <- runConversationItems(t.Context(), nil, producer, consumer)
	}()

	select {
	case <-cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("producer did not observe consumer cancellation")
	}
	select {
	case err := <-result:
		t.Fatalf("runConversationItems returned before its producer exited: %v", err)
	default:
	}
	close(releaseProducer)
	select {
	case err := <-result:
		assert.ErrorIs(t, err, consumerErr)
	case <-time.After(time.Second):
		t.Fatal("runConversationItems did not return after its producer exited")
	}
	select {
	case <-producerDone:
	default:
		t.Fatal("producer was not joined before runConversationItems returned")
	}
}

func TestStream_ConversationsCB(t *testing.T) {
	ctrl := gomock.NewController(t)
	cs := New(mock_client.NewMockSlack(ctrl), network.NoLimits)
	proc := mock_processor.NewMockConversations(ctrl)
	items := make([]structures.EntityItem, 100)
	items[0] = structures.EntityItem{Id: "not-a-valid-slack-link"}
	for i := 1; i < len(items); i++ {
		items[i] = structures.EntityItem{Id: "C12345678"}
	}

	done := make(chan error, 1)
	go func() {
		done <- cs.ConversationsCB(t.Context(), proc, items, func(Result) error { return nil })
	}()
	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("ConversationsCB did not return after a fatal item")
	}
}

func TestStream_Conversations(t *testing.T) {
	threadItem := structures.EntityItem{Id: "CTM1:1610000000.000000"}
	threadChannel := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
	threadMessages := []slack.Message{{Msg: slack.Msg{
		Channel:         "CTM1",
		Timestamp:       "1610000000.000000",
		ThreadTimestamp: "1610000000.000000",
	}}}

	t.Run("fatal thread result cancels queued work", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		cs := New(cl, network.NoLimits)
		cs.chanCache.set("CTM1", threadChannel)
		cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return(threadMessages, false, "", nil).AnyTimes()
		proc.EXPECT().ChannelInfo(gomock.Any(), gomock.Any(), "1610000000.000000").Return(nil).AnyTimes()
		proc.EXPECT().ThreadMessages(gomock.Any(), "CTM1", gomock.Any(), true, true, threadMessages).Return(assert.AnError).AnyTimes()

		items := make(chan structures.EntityItem, threadChanSz+1)
		for range threadChanSz + 1 {
			items <- threadItem
		}
		close(items)

		done := make(chan error, 1)
		go func() { done <- cs.Conversations(t.Context(), proc, items) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, assert.AnError)
		case <-time.After(time.Second):
			t.Fatal("Conversations did not join cancelled workers")
		}
	})

	t.Run("callback failure cancels and joins producers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		callbackErr := errors.New("callback failed")
		cs := New(cl, network.NoLimits, OptResultFn(func(Result) error { return callbackErr }))
		cs.chanCache.set("CTM1", threadChannel)
		cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return(threadMessages, false, "", nil).AnyTimes()
		proc.EXPECT().ChannelInfo(gomock.Any(), gomock.Any(), "1610000000.000000").Return(nil).AnyTimes()
		proc.EXPECT().ThreadMessages(gomock.Any(), "CTM1", gomock.Any(), true, true, threadMessages).Return(nil).AnyTimes()

		items := make(chan structures.EntityItem, threadChanSz+1)
		for range threadChanSz + 1 {
			items <- threadItem
		}
		close(items)

		done := make(chan error, 1)
		go func() { done <- cs.Conversations(t.Context(), proc, items) }()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, callbackErr)
		case <-time.After(time.Second):
			t.Fatal("Conversations did not join producers after callback failure")
		}
	})

	t.Run("external cancellation returns cause", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cs := New(mock_client.NewMockSlack(ctrl), network.NoLimits)
		proc := mock_processor.NewMockConversations(ctrl)
		ctx, cancel := context.WithCancelCause(t.Context())
		cause := errors.New("caller stopped")
		cancel(cause)

		err := cs.Conversations(ctx, proc, make(chan structures.EntityItem))
		assert.ErrorIs(t, err, cause)
	})

	t.Run("nonfatal channel error continues processing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		cs := New(cl, network.NoLimits)
		cl.EXPECT().GetConversationInfoContext(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, in *slack.GetConversationInfoInput) (*slack.Channel, error) {
				if in.ChannelID == "CMISSING" {
					return nil, slack.SlackErrorResponse{Err: errChanNotFound.Error()}
				}
				return threadChannel, nil
			}).AnyTimes()
		cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return(threadMessages, false, "", nil)
		proc.EXPECT().ChannelInfo(gomock.Any(), threadChannel, "1610000000.000000").Return(nil)
		proc.EXPECT().ThreadMessages(gomock.Any(), "CTM1", gomock.Any(), true, true, threadMessages).Return(nil)

		items := make(chan structures.EntityItem, 2)
		items <- structures.EntityItem{Id: "CMISSING:1610000000.000000"}
		items <- threadItem
		close(items)

		assert.NoError(t, cs.Conversations(t.Context(), proc, items))
	})
}

func Test_procChanMsg(t *testing.T) {
	type args struct {
		ctx context.Context
		// proc    processor.Conversations // supplied by test
		threadC chan request
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
		checkFn  func(t *testing.T, threadC <-chan request, mm []slack.Message)
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
		{
			name: "thread request carries parent message",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan request, 1),
				channel: TestChannel,
				isLast:  true,
				mm: []slack.Message{{Msg: slack.Msg{
					Channel:         TestChannel.ID,
					Timestamp:       "1577694990.000400",
					ThreadTimestamp: "1577694990.000400",
					LatestReply:     "1638784627.000300",
					ReplyCount:      3,
					Text:            "thread parent",
				}}},
			},
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 1, true, gomock.Any()).Times(1)
			},
			checkFn: func(t *testing.T, threadC <-chan request, mm []slack.Message) {
				t.Helper()
				parent := mm[0]
				mm[0].Text = "mutated after enqueue"
				req := <-threadC
				assert.Equal(t, TestChannel.ID, req.sl.Channel)
				assert.Equal(t, parent.ThreadTimestamp, req.sl.ThreadTS)
				if assert.NotNil(t, req.parent) {
					assert.Equal(t, parent, *req.parent)
				}
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
			if tt.checkFn != nil {
				tt.checkFn(t, tt.args.threadC, tt.args.mm)
			}
		})
	}

	t.Run("cancellation unblocks thread output", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mp := mock_processor.NewMockConversations(ctrl)
		mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 1, true, threadedMsg).Return(nil)
		ctx, cancel := context.WithCancelCause(t.Context())
		cause := errors.New("stop thread routing")
		cancel(cause)

		got, err := (&Stream{}).procChanMsg(ctx, mp, make(chan request), TestChannel, true, threadedMsg)
		assert.Equal(t, 1, got)
		assert.ErrorIs(t, err, cause)
	})
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
	fileMessages := []slack.Message{
		{
			Msg: slack.Msg{
				Channel:         "CTM1",
				Timestamp:       "1610000000.000000",
				ThreadTimestamp: "1610000000.000000",
				Files: []slack.File{
					{ID: "FILE_1", Name: "file1"},
					{ID: "FILE_2", Name: "file2"},
				},
			},
		},
		{
			Msg: slack.Msg{
				Channel:         "CTM1",
				Timestamp:       "1610000000.000001",
				ThreadTimestamp: "1610000000.000000",
				Files: []slack.File{
					{ID: "FILE_3", Name: "file1"},
					{ID: "FILE_4", Name: "file2"},
				},
			},
		},
		{
			Msg: slack.Msg{
				Channel:         "CTM1",
				Timestamp:       "1610000000.000002",
				ThreadTimestamp: "1610000000.000000",
				Files: []slack.File{
					{ID: "FILE_5", Name: "file5"},
					{ID: "FILE_6", Name: "file6"},
				},
			},
		},
	}
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
		{
			"all files from messages are collected",
			args{
				t.Context(),
				fixtures.DummyChannel("CTM1"),
				fileMessages[0].ThreadTimestamp,
				false,
				true,
				fileMessages,
			},
			func(mp *mock_processor.MockConversations) {
				channel := fixtures.DummyChannel("CTM1")
				mp.EXPECT().
					ThreadMessages(gomock.Any(), "CTM1", fileMessages[0], false, true, fileMessages).
					Return(nil)
				mp.EXPECT().
					Files(gomock.Any(), channel, fileMessages[1], fileMessages[1].Files).
					Return(nil)
				mp.EXPECT().
					Files(gomock.Any(), channel, fileMessages[2], fileMessages[2].Files).
					Return(nil)
			},
			false,
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

func TestStream_procChannelUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	cl := mock_client.NewMockSlack(ctrl)
	proc := mock_processor.NewMockConversations(ctrl)
	cs := New(cl, network.NoLimits)

	gomock.InOrder(
		cl.EXPECT().
			GetUsersInConversationContext(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
				assert.Empty(t, params.Cursor)
				return []string{"U2", "U1"}, "next", nil
			}),
		cl.EXPECT().
			GetUsersInConversationContext(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
				assert.Equal(t, "next", params.Cursor)
				return []string{"U3", "U2"}, "", nil
			}),
	)
	proc.EXPECT().
		ChannelUsers(gomock.Any(), "C1", "", []string{"U1", "U2", "U3"}).
		Return(nil)

	got, err := cs.procChannelUsers(t.Context(), proc, "C1", "")
	assert.NoError(t, err)
	assert.Equal(t, []string{"U1", "U2", "U3"}, got)

	// A second request should use the deduplicated cached value without calling
	// either the Slack API or the processor again.
	got, err = cs.procChannelUsers(t.Context(), proc, "C1", "")
	assert.NoError(t, err)
	assert.Equal(t, []string{"U1", "U2", "U3"}, got)
}

func Test_uniqueStrings(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		input []string
		want  []string
	}{
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "single element",
			input: []string{"a"},
			want:  []string{"a"},
		},
		{
			name:  "multiple unique elements",
			input: []string{"b", "a", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "multiple duplicate elements",
			input: []string{"b", "a", "c", "a", "b"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "all duplicates",
			input: []string{"a", "a", "a"},
			want:  []string{"a"},
		},
		{
			name:  "mixed case",
			input: []string{"A", "a", "B", "b"},
			want:  []string{"A", "B", "a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueStrings(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
