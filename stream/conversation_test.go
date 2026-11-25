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
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("enumerated channel and overlapping direct thread keep API calls", func(t *testing.T) {
		run := func(enabled bool) {
			ctrl := gomock.NewController(t)
			cl := mock_client.NewMockSlack(ctrl)
			proc := mock_processor.NewMockConversations(ctrl)
			channels := mock_processor.NewMockChannels(ctrl)
			owner := slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
			root := slack.Message{Msg: slack.Msg{Channel: "CTM1", Timestamp: "1610000000.000000", ThreadTimestamp: "1610000000.000000", LatestReply: "1610000001.000000", ReplyCount: 1}}
			opts := []Option{}
			if enabled {
				opts = append(opts, OptProgress(func(ProgressEvent) {}))
			}
			cs := New(cl, network.NoLimits, opts...)
			cs.chanCache.set(owner.ID, &owner)
			cl.EXPECT().GetConversationsContext(gomock.Any(), gomock.Any()).Return([]slack.Channel{owner}, "", nil).Times(1)
			channels.EXPECT().Channels(gomock.Any(), []slack.Channel{owner}).Return(nil).Times(1)
			proc.EXPECT().ChannelInfo(gomock.Any(), &owner, gomock.Any()).Return(nil).Times(2)
			cl.EXPECT().GetUsersInConversationContext(gomock.Any(), gomock.Any()).Return(nil, "", nil).Times(1)
			proc.EXPECT().ChannelUsers(gomock.Any(), owner.ID, "", []string(nil)).Return(nil).Times(1)
			cl.EXPECT().GetConversationHistoryContext(gomock.Any(), gomock.Any()).Return(&slack.GetConversationHistoryResponse{SlackResponse: slack.SlackResponse{Ok: true}, Messages: []slack.Message{root}}, nil).Times(1)
			proc.EXPECT().Messages(gomock.Any(), owner.ID, 1, true, []slack.Message{root}).Return(nil).Times(1)
			cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return([]slack.Message{root}, false, "", nil).Times(2)
			proc.EXPECT().ThreadMessages(gomock.Any(), owner.ID, root, gomock.Any(), true, []slack.Message{root}).Return(nil).Times(2)
			require.NoError(t, cs.ListChannels(t.Context(), channels, &slack.GetConversationsParameters{}))
			items := make(chan structures.EntityItem, 2)
			items <- structures.EntityItem{Id: owner.ID}
			items <- structures.EntityItem{Id: owner.ID + ":" + root.Timestamp}
			close(items)
			require.NoError(t, cs.Conversations(t.Context(), proc, items))
		}
		run(false)
		run(true)
	})
	t.Run("cancellation leaves queued jobs unresolved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		var mu sync.Mutex
		var events []ProgressEvent
		queued := make(chan struct{})
		started := make(chan struct{})
		count := 0
		cs := New(cl, network.NoLimits, OptProgress(func(e ProgressEvent) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)
			if e.Kind == ProgressThreadQueued {
				count++
				if count == 2 {
					close(queued)
				}
			}
		}))
		channel := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
		cs.chanCache.set("CTM1", channel)
		proc.EXPECT().ChannelInfo(gomock.Any(), channel, gomock.Any()).Return(nil).AnyTimes()
		cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _ *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
			close(started)
			<-ctx.Done()
			return nil, false, "", ctx.Err()
		}).Times(1)
		items := make(chan structures.EntityItem, 2)
		items <- structures.EntityItem{Id: "CTM1:1610000000.000000"}
		items <- structures.EntityItem{Id: "CTM1:1610000001.000000"}
		close(items)
		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- cs.Conversations(ctx, proc, items) }()
		<-started
		<-queued
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e.Kind == ProgressThreadDone {
				require.Error(t, e.Err, "cancelled job reported success")
			}
		}
	})
	t.Run("empty reply exhaustion is identical with progress", func(t *testing.T) {
		run := func(enabled bool) (int, error, []ProgressEvent) {
			ctrl := gomock.NewController(t)
			cl := mock_client.NewMockSlack(ctrl)
			proc := mock_processor.NewMockConversations(ctrl)
			var mu sync.Mutex
			var events []ProgressEvent
			calls := 0
			opts := []Option{}
			if enabled {
				opts = append(opts, OptProgress(func(e ProgressEvent) { mu.Lock(); events = append(events, e); mu.Unlock() }))
			}
			cs := New(cl, network.NoLimits, opts...)
			channel := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
			cs.chanCache.set("CTM1", channel)
			proc.EXPECT().ChannelInfo(gomock.Any(), channel, "1610000000.000000").Return(nil)
			cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
				calls++
				return nil, false, "", nil
			}).AnyTimes()
			items := make(chan structures.EntityItem, 1)
			items <- structures.EntityItem{Id: "CTM1:1610000000.000000"}
			close(items)
			err := cs.Conversations(t.Context(), proc, items)
			return calls, err, events
		}
		callsOff, errOff, _ := run(false)
		callsOn, errOn, events := run(true)
		require.Error(t, errOff)
		require.Error(t, errOn)
		require.Equal(t, callsOff, callsOn)
		require.Positive(t, callsOn)
		failed := false
		for _, e := range events {
			if e.Kind == ProgressThreadDone && e.Err != nil {
				failed = true
			}
		}
		require.True(t, failed)
	})
	t.Run("noncritical direct thread channel skips are resolved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		var mu sync.Mutex
		var events []ProgressEvent
		cs := New(cl, network.NoLimits, OptProgress(func(e ProgressEvent) { mu.Lock(); events = append(events, e); mu.Unlock() }))
		cl.EXPECT().GetConversationInfoContext(gomock.Any(), gomock.Any()).Return(nil, slack.SlackErrorResponse{Err: "channel_not_found"})
		items := make(chan structures.EntityItem, 1)
		items <- structures.EntityItem{Id: "CTM1:1610000000.000000"}
		close(items)
		require.NoError(t, cs.Conversations(t.Context(), proc, items))
		found := false
		for _, e := range events {
			if e.Kind == ProgressThreadDone {
				found = true
				require.NoError(t, e.Err)
			}
		}
		require.True(t, found)
	})
	t.Run("progress preserves direct skip calls and job outcomes", func(t *testing.T) {
		run := func(enabled bool) (int, []ProgressEvent) {
			ctrl := gomock.NewController(t)
			cl := mock_client.NewMockSlack(ctrl)
			proc := mock_processor.NewMockConversations(ctrl)
			var mu sync.Mutex
			var events []ProgressEvent
			calls := 0
			opts := []Option{OptSkipThreadFunc(func(context.Context, string, string, int) bool { return true })}
			if enabled {
				opts = append(opts, OptProgress(func(e ProgressEvent) { mu.Lock(); events = append(events, e); mu.Unlock() }))
			}
			cs := New(cl, network.NoLimits, opts...)
			channel := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
			cs.chanCache.set("CTM1", channel)
			proc.EXPECT().ChannelInfo(gomock.Any(), channel, "1610000000.000000").Return(nil).Times(2)
			cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
				calls++
				return []slack.Message{{Msg: slack.Msg{Timestamp: "1610000000.000000"}}}, false, "", nil
			}).Times(2)
			items := make(chan structures.EntityItem, 2)
			items <- structures.EntityItem{Id: "CTM1:1610000000.000000"}
			items <- structures.EntityItem{Id: "CTM1:1610000000.000000"}
			close(items)
			require.NoError(t, cs.Conversations(t.Context(), proc, items))
			return calls, events
		}
		without, _ := run(false)
		with, events := run(true)
		require.Equal(t, without, with)
		require.Equal(t, 2, with)
		queued, done := 0, 0
		for _, e := range events {
			if e.Kind == ProgressThreadQueued {
				queued++
			}
			if e.Kind == ProgressThreadDone {
				done++
				require.NoError(t, e.Err)
			}
		}
		require.Equal(t, 2, queued)
		require.Equal(t, 2, done)
	})
	t.Run("canvas discovery timeout cancels queued channels", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		cl := mock_client.NewMockSlack(ctrl)
		proc := mock_processor.NewMockConversations(ctrl)
		cm := mock_processor.NewMockCanvasMessenger(ctrl)
		owner := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "COWNER"}}, Properties: &slack.Properties{Canvas: slack.Canvas{FileId: "FCANVAS"}}}
		cl.EXPECT().GetConversationInfoContext(gomock.Any(), gomock.Any()).Return(owner, nil).AnyTimes()
		cl.EXPECT().GetUsersInConversationContext(gomock.Any(), gomock.Any()).Return(nil, "", nil).AnyTimes()
		proc.EXPECT().ChannelInfo(gomock.Any(), owner, "").Return(nil).AnyTimes()
		proc.EXPECT().ChannelUsers(gomock.Any(), owner.ID, "", []string(nil)).Return(nil).AnyTimes()
		cl.EXPECT().GetFileInfoContext(gomock.Any(), "FCANVAS", 0, 1).Return(&slack.File{ID: "FCANVAS"}, nil, nil, nil)
		proc.EXPECT().Files(gomock.Any(), owner, slack.Message{}, []slack.File{{ID: "FCANVAS"}}).Return(nil)
		discoveryErr := fmt.Errorf("HTTP request timed out: %w", context.DeadlineExceeded)
		cs := New(&canvasSlack{Slack: cl, supported: true, err: discoveryErr}, network.NoLimits)
		items := make(chan structures.EntityItem, msgChanSz+2)
		for range msgChanSz + 2 {
			items <- structures.EntityItem{Id: owner.ID}
		}
		close(items)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- cs.Conversations(ctx, &canvasConversations{proc, cm}, items) }()
		select {
		case err := <-done:
			require.ErrorIs(t, err, discoveryErr)
			require.NoError(t, ctx.Err(), "failure must not depend on outer cancellation")
		case <-time.After(3 * time.Second):
			cancel()
			<-done
			t.Fatal("Conversations did not cancel and join its pipeline")
		}
	})

	threadItem := structures.EntityItem{Id: "CTM1:1610000000.000000"}
	threadChannel := &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "CTM1"}}}
	threadMessages := []slack.Message{{Msg: slack.Msg{
		Channel:         "CTM1",
		Timestamp:       "1610000000.000000",
		ThreadTimestamp: "1610000000.000000",
	}}}

	t.Run("persistent invalid auth fails conversation backup", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			cl := mock_client.NewMockSlack(ctrl)
			proc := mock_processor.NewMockConversations(ctrl)
			cs := New(cl, network.NoLimits)
			cs.chanCache.set("CTM1", threadChannel)
			original := fmt.Errorf("request: %w", slack.SlackErrorResponse{Err: "invalid_auth"})
			cl.EXPECT().GetConversationRepliesContext(gomock.Any(), gomock.Any()).Return(nil, false, "", original).Times(3)
			proc.EXPECT().ChannelInfo(gomock.Any(), gomock.Any(), "1610000000.000000").Return(nil)
			items := make(chan structures.EntityItem, 1)
			items <- threadItem
			close(items)
			err := cs.Conversations(context.Background(), proc, items)
			assert.ErrorIs(t, err, original)
			assert.ErrorIs(t, err, &network.ErrRetryFailed{})
		})
	})

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
		var mu sync.Mutex
		var events []ProgressEvent
		queued := make(chan struct{})
		doneCount := 0
		cs := New(cl, network.NoLimits, OptProgress(func(e ProgressEvent) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, e)
			if e.Kind == ProgressThreadDone {
				doneCount++
				if doneCount == 2 {
					close(queued)
				}
			}
		}), OptResultFn(func(Result) error { <-queued; return callbackErr }))
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
		case <-time.After(3 * time.Second):
			t.Fatal("Conversations did not join producers after callback failure")
		}
		mu.Lock()
		defer mu.Unlock()
		queuedCount, processed, failed := 0, 0, 0
		for _, e := range events {
			switch e.Kind {
			case ProgressResultQueued:
				queuedCount++
			case ProgressResultProcessed:
				processed++
				if e.Err != nil {
					failed++
				}
			}
		}
		require.GreaterOrEqual(t, failed, 2, "callback and drained result must fail")
		require.Equal(t, queuedCount, processed, "every reserved callback must be released")
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
		threadC chan ordinaryThreadRequest
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
	parentWithFile := threadedMsg[0]
	parentWithFile.Files = []slack.File{{ID: "F123", Name: "parent.txt"}}

	tests := []struct {
		name     string
		args     args
		skipFn   func(ctx context.Context, channelID, threadTS string, replyCount int) bool
		expectFn func(mp *mock_processor.MockConversations)
		checkFn  func(t *testing.T, threadC <-chan ordinaryThreadRequest, mm []slack.Message)
		want     int
		wantErr  bool
	}{
		{
			name: "empty messages slice",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan ordinaryThreadRequest),
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
				threadC: make(chan ordinaryThreadRequest),
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
				threadC: make(chan ordinaryThreadRequest),
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
				threadC: make(chan ordinaryThreadRequest),
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
				threadC: make(chan ordinaryThreadRequest),
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
			name: "skip complete thread but process parent files",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan ordinaryThreadRequest, 1),
				channel: TestChannel,
				isLast:  true,
				mm:      []slack.Message{parentWithFile},
			},
			skipFn: func(_ context.Context, _, _ string, _ int) bool { return true },
			expectFn: func(mp *mock_processor.MockConversations) {
				mp.EXPECT().Files(gomock.Any(), TestChannel, parentWithFile, parentWithFile.Files).Times(1)
				mp.EXPECT().Messages(gomock.Any(), TestChannel.ID, 0, true, []slack.Message{parentWithFile}).Times(1)
			},
			checkFn: func(t *testing.T, threadC <-chan ordinaryThreadRequest, _ []slack.Message) {
				assert.Empty(t, threadC)
			},
			want: 0,
		},
		{
			name: "do not skip incomplete thread",
			args: args{
				ctx:     t.Context(),
				threadC: make(chan ordinaryThreadRequest, 1),
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
				threadC: make(chan ordinaryThreadRequest, 1),
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
			checkFn: func(t *testing.T, threadC <-chan ordinaryThreadRequest, mm []slack.Message) {
				t.Helper()
				parent := mm[0]
				mm[0].Text = "mutated after enqueue"
				req := <-threadC
				assert.Equal(t, TestChannel.ID, req.fetch.channelID)
				assert.Equal(t, parent.ThreadTimestamp, req.fetch.threadTS)
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

		got, err := (&Stream{}).procChanMsg(ctx, mp, make(chan ordinaryThreadRequest), TestChannel, true, threadedMsg)
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
