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
	reqC := make(chan request, 1)
	reqC <- request{sl: &structures.SlackLink{Channel: "COWNER"}}
	close(reqC)
	threadC := make(chan request, 1)
	results := make(chan Result, 2)

	cs.channelWorker(t.Context(), mc, results, threadC, reqC)

	assert.Empty(t, threadC)
	require.Len(t, results, 1)
	result := <-results
	assert.Equal(t, RTChannel, result.Type)
	assert.NoError(t, result.Err)
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
	ignored := slack.Message{Msg: slack.Msg{SubType: "message", Timestamp: "151.000001"}}
	call := 0
	ms.EXPECT().
		GetConversationHistoryContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
			assert.Equal(t, "CCANVAS", params.ChannelID)
			assert.Equal(t, structures.FormatSlackTS(oldest), params.Oldest)
			assert.Equal(t, structures.FormatSlackTS(latest), params.Latest)
			assert.True(t, params.Inclusive)
			call++
			if call == 1 {
				assert.Empty(t, params.Cursor)
				resp := &slack.GetConversationHistoryResponse{
					SlackResponse: slack.SlackResponse{Ok: true},
					Messages:      []slack.Message{root, ignored},
					HasMore:       true,
				}
				resp.ResponseMetaData.NextCursor = "next"
				return resp, nil
			}
			assert.Equal(t, "next", params.Cursor)
			return &slack.GetConversationHistoryResponse{
				SlackResponse: slack.SlackResponse{Ok: true},
				Messages:      []slack.Message{ignored},
			}, nil
		}).
		Times(2)
	mc.EXPECT().
		Files(gomock.Any(), owner, gomock.Any(), []slack.File{{ID: "FATTACH"}}).
		DoAndReturn(func(_ context.Context, _ *slack.Channel, parent slack.Message, _ []slack.File) error {
			assert.Equal(t, "150.000001", parent.Timestamp)
			return nil
		})
	cm.EXPECT().
		CanvasMessages(gomock.Any(), "CCANVAS", 1, false, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ int, _ bool, messages []slack.Message) error {
			require.Len(t, messages, 1)
			assert.Equal(t, messages[0].Timestamp, messages[0].ThreadTimestamp)
			return nil
		})
	cm.EXPECT().
		CanvasMessages(gomock.Any(), "CCANVAS", 0, true, []slack.Message{}).
		Return(nil)

	cs := New(ms, network.NoLimits)
	threadC := make(chan request, 1)
	err := cs.canvasDiscussions(t.Context(), mc, cm, threadC, request{
		Oldest: oldest,
		Latest: latest,
	}, owner, "FCANVAS")
	require.NoError(t, err)
	require.Len(t, threadC, 1)
	req := <-threadC
	assert.Equal(t, requestCanvas, req.kind)
	assert.Equal(t, "CCANVAS", req.sl.Channel)
	assert.Equal(t, "150.000001", req.sl.ThreadTS)
	assert.Equal(t, oldest, req.Oldest)
	assert.Equal(t, latest, req.Latest)
	assert.Same(t, owner, req.canvas.owner)
	assert.Equal(t, "FCANVAS", req.canvas.fileID)
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
	threadC := make(chan request, len(roots))
	cm.EXPECT().
		CanvasMessages(gomock.Any(), "CHIDDEN", 1, true, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ int, _ bool, messages []slack.Message) error {
			require.Len(t, messages, 3)
			assert.Len(t, threadC, 0, "roots must be recorded before replies are queued")
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
	err := cs.procCanvasMsg(t.Context(), mc, cm, threadC, request{
		sl:     &structures.SlackLink{Channel: "CHIDDEN"},
		kind:   requestCanvas,
		canvas: &canvasRequest{owner: owner, fileID: "FHIDDEN"},
	}, true, roots)
	require.NoError(t, err)
	require.Len(t, threadC, 1)
	req := <-threadC
	assert.Equal(t, "1.0", req.sl.ThreadTS)
	assert.Equal(t, requestCanvas, req.kind)
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

func TestStream_threadWorker(t *testing.T) {
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
		wantResult   bool
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
			wantResult:   true,
			wantErr:      assert.AnError,
		},
		{
			name:       "canvas cancellation is fatal",
			apiErr:     context.Canceled,
			wantResult: true,
			wantErr:    context.Canceled,
		},
		{
			name:       "canvas deadline is fatal",
			apiErr:     context.DeadlineExceeded,
			wantResult: true,
			wantErr:    context.DeadlineExceeded,
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
			} else {
				ms.EXPECT().
					GetConversationRepliesContext(gomock.Any(), gomock.Any()).
					Return(messages, false, "", nil)
				cm.EXPECT().
					CanvasThreadMessages(gomock.Any(), "CHIDDEN", messages[0], true, messages).
					Return(tt.processorErr)
			}

			cs := New(ms, network.NoLimits)
			reqC := make(chan request, 1)
			reqC <- request{
				sl:   &structures.SlackLink{Channel: "CHIDDEN", ThreadTS: "1.0"},
				kind: requestCanvas,
				canvas: &canvasRequest{
					owner:  owner,
					fileID: "FHIDDEN",
				},
			}
			close(reqC)
			results := make(chan Result, 2)
			cs.threadWorker(t.Context(), &canvasConversations{
				Conversations:   mc,
				CanvasMessenger: cm,
			}, results, reqC)

			if !tt.wantResult {
				assert.Empty(t, results)
				return
			}
			require.Len(t, results, 1)
			result := <-results
			assert.ErrorIs(t, result.Err, tt.wantErr)
		})
	}
}
