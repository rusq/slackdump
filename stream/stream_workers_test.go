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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/internal/client"
	"github.com/rusq/slackdump/v4/internal/client/mock_client"
	"github.com/rusq/slackdump/v4/internal/fixtures"
	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/mocks/mock_processor"
	"github.com/rusq/slackdump/v4/processor"
)

type canvasConversations struct {
	processor.Conversations
	processor.CanvasMessenger
}

type canvasSlack struct {
	client.Slack
	supported bool
	roots     []slack.Message
	err       error
	calls     int
}

func (c *canvasSlack) CanvasSupported() bool {
	return c.supported
}

func (c *canvasSlack) CanvasThreadRoots(context.Context, string) ([]slack.Message, error) {
	c.calls++
	return c.roots, c.err
}

func TestStream_channelWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	ms := mock_client.NewMockSlack(ctrl)
	mc := mock_processor.NewMockConversations(ctrl)
	owner := &slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{ID: "COWNER"},
		},
		Properties: &slack.Properties{
			Canvas: slack.Canvas{FileId: "FCANVAS"},
		},
	}
	ms.EXPECT().
		GetConversationInfoContext(gomock.Any(), gomock.Any()).
		Return(owner, nil)
	ms.EXPECT().
		GetUsersInConversationContext(gomock.Any(), gomock.Any()).
		Return(nil, "", nil)
	mc.EXPECT().ChannelInfo(gomock.Any(), owner, "").Return(nil)
	mc.EXPECT().ChannelUsers(gomock.Any(), "COWNER", "", []string(nil)).Return(nil)
	ms.EXPECT().
		GetFileInfoContext(gomock.Any(), "FCANVAS", 0, 1).
		Return(&slack.File{ID: "FCANVAS"}, nil, nil, nil)
	mc.EXPECT().
		Files(gomock.Any(), owner, slack.Message{}, []slack.File{{ID: "FCANVAS"}}).
		Return(nil)
	ms.EXPECT().
		GetConversationHistoryContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
			assert.Equal(t, "COWNER", params.ChannelID, "legacy processors must not trigger hidden canvas history")
			return &slack.GetConversationHistoryResponse{
				SlackResponse: slack.SlackResponse{Ok: true},
			}, nil
		})
	mc.EXPECT().Messages(gomock.Any(), "COWNER", 0, true, []slack.Message(nil)).Return(nil)

	cs := New(ms, network.NoLimits)
	reqC := make(chan channelRequest, 1)
	reqC <- channelRequest{sl: structures.SlackLink{Channel: "COWNER"}}
	close(reqC)
	threadC := make(chan ordinaryThreadRequest, 1)
	canvasC := make(chan canvasThreadRequest, 1)
	canvasResultsC := make(chan canvasThreadResult, 1)
	results := make(chan Result, 2)

	cs.channelWorker(t.Context(), mc, results, threadC, canvasC, canvasResultsC, reqC)

	assert.Empty(t, threadC)
	require.Len(t, results, 1)
	result := <-results
	assert.Equal(t, RTChannel, result.Type)
	assert.NoError(t, result.Err)
}

func TestStream_channelWorker_canvasBypassesOrdinaryThreadBacklog(t *testing.T) {
	ctrl := gomock.NewController(t)
	ms := mock_client.NewMockSlack(ctrl)
	mc := mock_processor.NewMockConversations(ctrl)
	cm := mock_processor.NewMockCanvasMessenger(ctrl)
	owner := &slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{ID: "COWNER"},
		},
		Properties: &slack.Properties{
			Canvas: slack.Canvas{FileId: "FCANVAS"},
		},
	}
	root := slack.Message{Msg: slack.Msg{
		SubType:         structures.SubTypeDocumentCommentRoot,
		Timestamp:       "1.0",
		ThreadTimestamp: "1.0",
		ReplyCount:      1,
	}}
	reply := slack.Message{Msg: slack.Msg{
		Timestamp:       "2.0",
		ThreadTimestamp: root.Timestamp,
	}}
	canvasDone := make(chan struct{})
	ordinaryStarted := make(chan struct{})
	releaseOrdinary := make(chan struct{})
	ordinaryRoot := slack.Message{Msg: slack.Msg{Channel: "COLD", Timestamp: "10.0", ThreadTimestamp: "10.0"}}

	ms.EXPECT().GetConversationInfoContext(gomock.Any(), gomock.Any()).Return(owner, nil)
	ms.EXPECT().GetUsersInConversationContext(gomock.Any(), gomock.Any()).Return(nil, "", nil)
	ms.EXPECT().GetFileInfoContext(gomock.Any(), "FCANVAS", 0, 1).Return(&slack.File{ID: "FCANVAS"}, nil, nil, nil)
	ms.EXPECT().
		GetConversationRepliesContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
			if params.ChannelID == "COLD" {
				close(ordinaryStarted)
				<-releaseOrdinary
				return []slack.Message{ordinaryRoot}, false, "", nil
			}
			assert.Equal(t, "CCANVAS", params.ChannelID)
			return []slack.Message{root, reply}, false, "", nil
		}).
		Times(2)
	ms.EXPECT().
		GetConversationHistoryContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
			select {
			case <-canvasDone:
			default:
				t.Fatal("ordinary channel history started before canvas discussions completed")
			}
			return &slack.GetConversationHistoryResponse{SlackResponse: slack.SlackResponse{Ok: true}}, nil
		})

	mc.EXPECT().ChannelInfo(gomock.Any(), owner, "").Return(nil)
	mc.EXPECT().ChannelUsers(gomock.Any(), "COWNER", "", []string(nil)).Return(nil)
	mc.EXPECT().Files(gomock.Any(), owner, slack.Message{}, []slack.File{{ID: "FCANVAS"}}).Return(nil)
	mc.EXPECT().Messages(gomock.Any(), "COWNER", 0, true, []slack.Message(nil)).Return(nil)
	mc.EXPECT().ThreadMessages(gomock.Any(), "COLD", ordinaryRoot, false, true, []slack.Message{ordinaryRoot}).Return(nil)
	cm.EXPECT().CanvasMessages(gomock.Any(), "CCANVAS", 1, true, []slack.Message{root}).Return(nil)
	cm.EXPECT().
		CanvasThreadMessages(gomock.Any(), "CCANVAS", root, true, []slack.Message{root, reply}).
		DoAndReturn(func(context.Context, string, slack.Message, bool, []slack.Message) error {
			close(canvasDone)
			return nil
		})

	canvasClient := &canvasSlack{
		Slack:     ms,
		supported: true,
		roots:     []slack.Message{root},
	}
	cs := New(canvasClient, network.NoLimits)
	proc := &canvasConversations{Conversations: mc, CanvasMessenger: cm}
	threadC := make(chan ordinaryThreadRequest, 1)
	threadC <- ordinaryThreadRequest{fetch: threadFetchSpec{channelID: "COLD", threadTS: "10.0"}}
	canvasC := make(chan canvasThreadRequest, 1)
	canvasResultsC := make(chan canvasThreadResult, 1)
	threadWorkerDone := make(chan struct{})
	go func() {
		defer close(threadWorkerDone)
		cs.canvasThreadWorker(t.Context(), proc, canvasResultsC, canvasC)
	}()
	ordinaryWorkerDone := make(chan struct{})
	go func() {
		defer close(ordinaryWorkerDone)
		cs.threadWorker(t.Context(), proc, make(chan Result, 1), threadC)
	}()
	<-ordinaryStarted

	reqC := make(chan channelRequest, 1)
	reqC <- channelRequest{sl: structures.SlackLink{Channel: "COWNER"}}
	close(reqC)
	results := make(chan Result, 2)
	cs.channelWorker(t.Context(), proc, results, threadC, canvasC, canvasResultsC, reqC)
	close(canvasC)
	<-threadWorkerDone
	close(releaseOrdinary)
	close(threadC)
	<-ordinaryWorkerDone

	require.Len(t, results, 2)
	assert.Equal(t, RTCanvasThread, (<-results).Type)
	assert.NoError(t, (<-results).Err)
}

func TestStream_canvasFile(t *testing.T) {
	testChannel := fixtures.Load[[]*slack.Channel](fixtures.TestChannelsJSON)[0]
	type args struct {
		ctx context.Context
		// proc    processor.Conversations
		channel *slack.Channel
		fileId  string
	}
	tests := []struct {
		name      string
		fields    *Stream
		args      args
		expectFn  func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations)
		wantErr   bool
		wantAPI   bool
		wantFatal bool
	}{
		{
			name:   "file ID is empty",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: &slack.Channel{},
				fileId:  "",
			},
			wantErr: false,
		},
		{
			name:   "getfileinfocontext returns an error",
			fields: &Stream{},
			args: args{
				ctx:    t.Context(),
				fileId: "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().GetFileInfoContext(gomock.Any(), "F123456", 0, 1).Return(nil, nil, nil, errors.New("getfileinfocontext error"))
			},
			wantErr:   true,
			wantAPI:   true,
			wantFatal: false,
		},
		{
			name:   "file not found",
			fields: &Stream{},
			args: args{
				ctx:    t.Context(),
				fileId: "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().GetFileInfoContext(gomock.Any(), "F123456", 0, 1).Return(nil, nil, nil, nil)
			},
			wantErr:   true,
			wantAPI:   true,
			wantFatal: false,
		},
		{
			name:   "success",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: testChannel,
				fileId:  "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().
					GetFileInfoContext(gomock.Any(), "F123456", 0, 1).
					Return(&slack.File{ID: "F123456"}, nil, nil, nil)
				mc.EXPECT().
					Files(gomock.Any(), testChannel, slack.Message{}, []slack.File{{ID: "F123456"}}).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "processor returns an error",
			fields: &Stream{},
			args: args{
				ctx:     t.Context(),
				channel: testChannel,
				fileId:  "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().
					GetFileInfoContext(gomock.Any(), "F123456", 0, 1).
					Return(&slack.File{ID: "F123456"}, nil, nil, nil)
				mc.EXPECT().
					Files(gomock.Any(), testChannel, slack.Message{}, []slack.File{{ID: "F123456"}}).
					Return(assert.AnError)
			},
			wantErr:   true,
			wantFatal: true,
		},
		{
			name:   "context cancellation is fatal",
			fields: &Stream{},
			args: args{
				ctx:    t.Context(),
				fileId: "F123456",
			},
			expectFn: func(ms *mock_client.MockSlack, mc *mock_processor.MockConversations) {
				ms.EXPECT().GetFileInfoContext(gomock.Any(), "F123456", 0, 1).Return(nil, nil, nil, context.Canceled)
			},
			wantErr:   true,
			wantAPI:   true,
			wantFatal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mc := mock_processor.NewMockConversations(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(ms, mc)
			}
			cs := tt.fields
			cs.client = ms
			err := cs.canvasFile(tt.args.ctx, mc, tt.args.channel, tt.args.fileId)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stream.canvasFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantAPI, isAPIError(err))
			if err != nil {
				assert.Equal(t, tt.wantFatal, canvasErrorIsFatal(err))
			}
		})
	}
}

func TestStream_canvasDiscussions(t *testing.T) {
	ctrl := gomock.NewController(t)
	ms := mock_client.NewMockSlack(ctrl)
	mc := mock_processor.NewMockConversations(ctrl)
	cm := mock_processor.NewMockCanvasMessenger(ctrl)
	owner := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "COWNER"},
	}}
	oldest := time.Unix(100, 0)
	latest := time.Unix(200, 0)
	root := slack.Message{Msg: slack.Msg{
		SubType:    structures.SubTypeDocumentCommentRoot,
		Timestamp:  "150.000001",
		ReplyCount: 2,
		Files:      []slack.File{{ID: "FATTACH"}},
	}}
	before := slack.Message{Msg: slack.Msg{SubType: structures.SubTypeDocumentCommentRoot, Timestamp: "99.999999"}}
	after := slack.Message{Msg: slack.Msg{SubType: structures.SubTypeDocumentCommentRoot, Timestamp: "200.000001"}}
	canvasClient := &canvasSlack{
		Slack:     ms,
		supported: true,
		roots:     []slack.Message{before, root, after},
	}
	mc.EXPECT().
		Files(gomock.Any(), owner, gomock.Any(), []slack.File{{ID: "FATTACH"}}).
		DoAndReturn(func(_ context.Context, _ *slack.Channel, parent slack.Message, _ []slack.File) error {
			assert.Equal(t, "150.000001", parent.Timestamp)
			return nil
		})
	cm.EXPECT().
		CanvasMessages(gomock.Any(), "CCANVAS", 1, true, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ int, _ bool, messages []slack.Message) error {
			require.Len(t, messages, 1)
			assert.Equal(t, messages[0].Timestamp, messages[0].ThreadTimestamp)
			return nil
		})

	cs := New(canvasClient, network.NoLimits)
	threadC := make(chan canvasThreadRequest, 1)
	completed := make(chan canvasThreadResult, 1)
	results := make(chan Result, 1)
	captured := make(chan canvasThreadRequest, 1)
	go func() {
		req := <-threadC
		captured <- req
		completed <- canvasThreadResult{channelID: req.fetch.channelID, threadTS: req.fetch.threadTS}
	}()
	err := cs.canvasDiscussions(t.Context(), mc, cm, threadC, completed, results, channelRequest{
		oldest: oldest,
		latest: latest,
	}, owner, "FCANVAS")
	require.NoError(t, err)
	assert.Equal(t, 1, canvasClient.calls)
	req := <-captured
	assert.Equal(t, "CCANVAS", req.fetch.channelID)
	assert.Equal(t, "150.000001", req.fetch.threadTS)
	assert.Equal(t, oldest, req.fetch.oldest)
	assert.Equal(t, latest, req.fetch.latest)
	assert.Same(t, owner, req.owner)
	assert.Equal(t, "FCANVAS", req.fileID)
	result := <-results
	assert.Equal(t, RTCanvasThread, result.Type)
	assert.Equal(t, "CCANVAS", result.ChannelID)
	assert.Equal(t, "150.000001", result.ThreadTS)
	assert.True(t, result.IsLast)

	unsupported := New(&canvasSlack{Slack: ms}, network.NoLimits)
	err = unsupported.canvasDiscussions(t.Context(), mc, cm, threadC, completed, results, channelRequest{}, owner, "FCANVAS")
	require.ErrorIs(t, err, client.ErrOpNotSupported)
}

func TestStream_filterCanvasRoots(t *testing.T) {
	messages := []slack.Message{
		{Msg: slack.Msg{Timestamp: "100.000000"}},
		{Msg: slack.Msg{Timestamp: "150.000000"}},
		{Msg: slack.Msg{Timestamp: "200.000000"}},
	}
	req := channelRequest{oldest: time.Unix(100, 0), latest: time.Unix(200, 0)}

	inclusive := &Stream{inclusive: true}
	got, err := inclusive.filterCanvasRoots(messages, req)
	require.NoError(t, err)
	assert.Equal(t, messages, got)

	exclusive := &Stream{}
	got, err = exclusive.filterCanvasRoots(messages, req)
	require.NoError(t, err)
	assert.Equal(t, messages[1:2], got)

	resume := New(nil, network.NoLimits, OptInclusive(false), OptIncludeOlderCanvasRoots())
	got, err = resume.filterCanvasRoots(messages, req)
	require.NoError(t, err)
	assert.Equal(t, messages[:2], got, "resume includes roots before oldest but still excludes roots at an exclusive latest bound")

	_, err = inclusive.filterCanvasRoots([]slack.Message{{Msg: slack.Msg{Timestamp: "invalid"}}}, req)
	require.Error(t, err)
}

func TestStream_procCanvasMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	mc := mock_processor.NewMockConversations(ctrl)
	cm := mock_processor.NewMockCanvasMessenger(ctrl)
	owner := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "COWNER"},
	}}
	roots := []slack.Message{
		{Msg: slack.Msg{SubType: structures.SubTypeDocumentCommentRoot, Timestamp: "1.0", ReplyCount: 2}},
		{Msg: slack.Msg{SubType: structures.SubTypeDocumentCommentRoot, Timestamp: "2.0"}},
		{Msg: slack.Msg{SubType: structures.SubTypeDocumentCommentRoot, Timestamp: "3.0", ReplyCount: 3}},
		{Msg: slack.Msg{SubType: "message", Timestamp: "4.0", ReplyCount: 4}},
	}
	cm.EXPECT().
		CanvasMessages(gomock.Any(), "CHIDDEN", 1, true, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ int, _ bool, messages []slack.Message) error {
			require.Len(t, messages, 3)
			for _, message := range messages {
				assert.Equal(t, message.Timestamp, message.ThreadTimestamp)
			}
			return nil
		})

	cs := &Stream{
		skipCanvasThread: func(_ context.Context, channelID, threadTS string, replyCount int) bool {
			assert.Equal(t, "CHIDDEN", channelID)
			return threadTS == "3.0" && replyCount == 3
		},
	}
	threads, err := cs.procCanvasMsg(t.Context(), mc, cm, owner, "CHIDDEN", "FHIDDEN", time.Time{}, time.Time{}, true, roots)
	require.NoError(t, err)
	require.Len(t, threads, 1)
	assert.Equal(t, "1.0", threads[0].fetch.threadTS)
	assert.Equal(t, "CHIDDEN", threads[0].fetch.channelID)
}

func Test_procCanvasThreadMsg(t *testing.T) {
	ctrl := gomock.NewController(t)
	mc := mock_processor.NewMockConversations(ctrl)
	cm := mock_processor.NewMockCanvasMessenger(ctrl)
	owner := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "COWNER"},
	}}
	messages := []slack.Message{
		{Msg: slack.Msg{Timestamp: "1.0", Files: []slack.File{{ID: "FROOT"}}}},
		{Msg: slack.Msg{Timestamp: "2.0", Files: []slack.File{{ID: "FREPLY"}}}},
	}
	mc.EXPECT().
		Files(gomock.Any(), owner, messages[1], []slack.File{{ID: "FREPLY"}}).
		Return(nil)
	cm.EXPECT().
		CanvasThreadMessages(gomock.Any(), "CHIDDEN", gomock.Any(), true, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, parent slack.Message, _ bool, got []slack.Message) error {
			assert.Equal(t, "1.0", parent.ThreadTimestamp)
			assert.Equal(t, "1.0", got[0].ThreadTimestamp)
			return nil
		})

	require.NoError(t, procCanvasThreadMsg(t.Context(), mc, cm, owner, "CHIDDEN", true, messages))
}

func TestStream_canvasThreadWorker(t *testing.T) {
	owner := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "COWNER"},
	}}
	messages := []slack.Message{
		{Msg: slack.Msg{Timestamp: "1.0", ThreadTimestamp: "1.0"}},
		{Msg: slack.Msg{Timestamp: "2.0", ThreadTimestamp: "1.0"}},
	}

	tests := []struct {
		name         string
		apiErr       error
		processorErr error
		wantErr      error
	}{
		{
			name: "canvas success emits no ordinary result",
		},
		{
			name:   "canvas API failure is warning only",
			apiErr: assert.AnError,
		},
		{
			name:   "canvas thread not found is warning only",
			apiErr: slack.SlackErrorResponse{Err: "thread_not_found"},
		},
		{
			name:         "canvas processor failure is fatal",
			processorErr: assert.AnError,
			wantErr:      assert.AnError,
		},
		{
			name:    "canvas cancellation is fatal",
			apiErr:  context.Canceled,
			wantErr: context.Canceled,
		},
		{
			name:    "canvas deadline is fatal",
			apiErr:  context.DeadlineExceeded,
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ms := mock_client.NewMockSlack(ctrl)
			mc := mock_processor.NewMockConversations(ctrl)
			cm := mock_processor.NewMockCanvasMessenger(ctrl)
			if tt.apiErr != nil {
				ms.EXPECT().
					GetConversationRepliesContext(gomock.Any(), gomock.Any()).
					Return(nil, false, "", tt.apiErr).
					MinTimes(1)
				if !errors.Is(tt.apiErr, context.Canceled) && !errors.Is(tt.apiErr, context.DeadlineExceeded) {
					fallback := messages[0]
					cm.EXPECT().
						CanvasThreadMessages(gomock.Any(), "CHIDDEN", fallback, true, []slack.Message{fallback}).
						Return(nil)
				}
			} else {
				ms.EXPECT().
					GetConversationRepliesContext(gomock.Any(), gomock.Any()).
					Return(messages, false, "", nil)
				cm.EXPECT().
					CanvasThreadMessages(gomock.Any(), "CHIDDEN", messages[0], true, messages).
					Return(tt.processorErr)
			}

			cs := New(ms, network.NoLimits)
			reqC := make(chan canvasThreadRequest, 1)
			parent := messages[0]
			reqC <- canvasThreadRequest{
				fetch:  threadFetchSpec{channelID: "CHIDDEN", threadTS: "1.0"},
				root:   parent,
				owner:  owner,
				fileID: "FHIDDEN",
			}
			close(reqC)
			completed := make(chan canvasThreadResult, 1)
			cs.canvasThreadWorker(t.Context(), &canvasConversations{
				Conversations:   mc,
				CanvasMessenger: cm,
			}, completed, reqC)

			require.Len(t, completed, 1)
			result := <-completed
			assert.Equal(t, "CHIDDEN", result.channelID)
			assert.Equal(t, "1.0", result.threadTS)
			assert.ErrorIs(t, result.err, tt.wantErr)
		})
	}
}

func TestStream_processCanvasThread(t *testing.T) {
	ctrl := gomock.NewController(t)
	ms := mock_client.NewMockSlack(ctrl)
	mc := mock_processor.NewMockConversations(ctrl)
	cm := mock_processor.NewMockCanvasMessenger(ctrl)
	owner := &slack.Channel{GroupConversation: slack.GroupConversation{
		Conversation: slack.Conversation{ID: "COWNER"},
	}}
	root := slack.Message{Msg: slack.Msg{
		Timestamp:       "1.0",
		ThreadTimestamp: "1.0",
		ReplyCount:      2,
	}}
	firstReply := slack.Message{Msg: slack.Msg{
		Timestamp:       "2.0",
		ThreadTimestamp: root.Timestamp,
	}}

	first := ms.EXPECT().
		GetConversationRepliesContext(gomock.Any(), gomock.Any()).
		Return([]slack.Message{root, firstReply}, true, "next", nil)
	ms.EXPECT().
		GetConversationRepliesContext(gomock.Any(), gomock.Any()).
		Return(nil, false, "", assert.AnError).
		After(first).
		MinTimes(1)
	cm.EXPECT().
		CanvasThreadMessages(gomock.Any(), "CHIDDEN", root, false, []slack.Message{root, firstReply}).
		Return(nil)
	cm.EXPECT().
		CanvasThreadMessages(gomock.Any(), "CHIDDEN", root, true, []slack.Message{root}).
		Return(nil)

	cs := New(ms, network.NoLimits)
	req := canvasThreadRequest{
		fetch:  threadFetchSpec{channelID: "CHIDDEN", threadTS: root.Timestamp},
		root:   root,
		owner:  owner,
		fileID: "FHIDDEN",
	}
	require.NoError(t, cs.processCanvasThread(t.Context(), &canvasConversations{
		Conversations:   mc,
		CanvasMessenger: cm,
	}, req))
}
