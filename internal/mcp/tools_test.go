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

package mcp

import (
	"context"
	"errors"
	"iter"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

// seqOf returns an iter.Seq2[slack.Message, error] that yields the given
// messages in order.
func seqOf(msgs ...slack.Message) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		for _, m := range msgs {
			if !yield(m, nil) {
				return
			}
		}
	}
}

// seqErr returns an iter.Seq2 that immediately yields an error.
func seqErr(err error) iter.Seq2[slack.Message, error] {
	return func(yield func(slack.Message, error) bool) {
		yield(slack.Message{}, err)
	}
}

// isErrorResult returns true when the result carries IsError=true.
func isErrorResult(r *mcplib.CallToolResult) bool {
	return r != nil && r.IsError
}

// firstText returns the text of the first TextContent in the result.
func firstText(t *testing.T, r *mcplib.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, r.Content, "result has no content")
	txt, ok := r.Content[0].(mcplib.TextContent)
	require.True(t, ok, "first content item is not TextContent")
	return txt.Text
}

// ─── handleListChannels ───────────────────────────────────────────────────────

func TestServer_handleListChannels(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string // substring expected in first text content
	}{
		{
			name: "returns channel list as JSON",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{
					{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}, Name: "general"}, IsChannel: true},
					{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C2"}, Name: "random"}, IsChannel: true},
				}, nil)
			},
			wantText: "C1",
		},
		{
			name: "empty list returns empty JSON array",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{}, nil)
			},
			wantText: "[]",
		},
		{
			name: "ErrNotSupported returns informational text",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Channels(gomock.Any()).Return(nil, source.ErrNotSupported)
			},
			wantText: "not support",
		},
		{
			name: "generic error returns error result",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Channels(gomock.Any()).Return(nil, errors.New("disk failure"))
			},
			wantIsError: true,
			wantText:    "disk failure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleListChannels(t.Context(), mcplib.CallToolRequest{})
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleListChannels) })
	t.Run("replacement waits for source read", func(t *testing.T) {
		checkHandlerRead(t, func(m *mock_source.MockSourceResumeCloser, block func()) {
			m.EXPECT().Channels(gomock.Any()).DoAndReturn(func(context.Context) ([]slack.Channel, error) { block(); return nil, nil })
		}, (*Server).handleListChannels, map[string]any{"channel_id": "C1"}, false)
	})

}

// ─── handleGetChannel ─────────────────────────────────────────────────────────

func TestServer_handleGetChannel(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string
	}{
		{
			name:        "missing channel_id returns error result",
			args:        nil,
			setup:       func(m *mock_source.MockSourceResumeCloser) {},
			wantIsError: true,
			wantText:    "channel_id",
		},
		{
			name: "returns channel JSON",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().ChannelInfo(gomock.Any(), "C1").Return(
					&slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C1"}, Name: "general"}},
					nil,
				)
			},
			wantText: "C1",
		},
		{
			name: "ErrNotFound returns informational text",
			args: map[string]any{"channel_id": "C999"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().ChannelInfo(gomock.Any(), "C999").Return(nil, source.ErrNotFound)
			},
			wantText: "C999",
		},
		{
			name: "ErrNotSupported returns informational text",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().ChannelInfo(gomock.Any(), "C1").Return(nil, source.ErrNotSupported)
			},
			wantText: "not support",
		},
		{
			name: "generic error returns error result",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().ChannelInfo(gomock.Any(), "C1").Return(nil, errors.New("io error"))
			},
			wantIsError: true,
			wantText:    "io error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleGetChannel(t.Context(), toolReq(tt.args))
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleGetChannel) })
	t.Run("replacement waits for source read", func(t *testing.T) {
		checkHandlerRead(t, func(m *mock_source.MockSourceResumeCloser, block func()) {
			m.EXPECT().ChannelInfo(gomock.Any(), "C1").DoAndReturn(func(context.Context, string) (*slack.Channel, error) { block(); return &slack.Channel{}, nil })
		}, (*Server).handleGetChannel, map[string]any{"channel_id": "C1"}, false)
	})

}

// ─── handleListUsers ──────────────────────────────────────────────────────────

func TestServer_handleListUsers(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string
	}{
		{
			name: "returns user list as JSON",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Users(gomock.Any()).Return([]slack.User{
					{ID: "U1", Name: "alice", RealName: "Alice A"},
				}, nil)
			},
			wantText: "U1",
		},
		{
			name: "ErrNotSupported returns informational text",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Users(gomock.Any()).Return(nil, source.ErrNotSupported)
			},
			wantText: "not support",
		},
		{
			name: "generic error returns error result",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().Users(gomock.Any()).Return(nil, errors.New("read err"))
			},
			wantIsError: true,
			wantText:    "read err",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleListUsers(t.Context(), mcplib.CallToolRequest{})
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleListUsers) })
	t.Run("replacement waits for source read", func(t *testing.T) {
		checkHandlerRead(t, func(m *mock_source.MockSourceResumeCloser, block func()) {
			m.EXPECT().Users(gomock.Any()).DoAndReturn(func(context.Context) ([]slack.User, error) { block(); return nil, nil })
		}, (*Server).handleListUsers, map[string]any{"channel_id": "C1"}, false)
	})

}

// ─── handleGetMessages ────────────────────────────────────────────────────────

func TestServer_handleGetMessages(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string
	}{
		{
			name:        "missing channel_id returns error result",
			args:        nil,
			setup:       func(m *mock_source.MockSourceResumeCloser) {},
			wantIsError: true,
			wantText:    "channel_id",
		},
		{
			name: "returns messages as JSON",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C1").Return(seqOf(
					slack.Message{Msg: slack.Msg{Timestamp: "1000.000001", Text: "hello", User: "U1"}},
					slack.Message{Msg: slack.Msg{Timestamp: "1001.000001", Text: "world", User: "U2"}},
				), nil)
			},
			wantText: "hello",
		},
		{
			name: "limit is respected",
			args: map[string]any{"channel_id": "C1", "limit": float64(1)},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C1").Return(seqOf(
					slack.Message{Msg: slack.Msg{Timestamp: "1000.000001", Text: "first", User: "U1"}},
					slack.Message{Msg: slack.Msg{Timestamp: "1001.000001", Text: "second", User: "U2"}},
				), nil)
			},
			wantText: "first",
		},
		{
			name: "after_ts filter skips old messages",
			args: map[string]any{"channel_id": "C1", "after_ts": "1000.000001"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C1").Return(seqOf(
					slack.Message{Msg: slack.Msg{Timestamp: "1000.000001", Text: "old"}},
					slack.Message{Msg: slack.Msg{Timestamp: "1001.000001", Text: "new"}},
				), nil)
			},
			wantText: "new",
		},
		{
			name: "ErrNotFound returns informational text",
			args: map[string]any{"channel_id": "C999"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C999").Return(nil, source.ErrNotFound)
			},
			wantText: "C999",
		},
		{
			name: "ErrNotSupported returns informational text",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C1").Return(nil, source.ErrNotSupported)
			},
			wantText: "not support",
		},
		{
			name: "iterator error returns error result",
			args: map[string]any{"channel_id": "C1"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllMessages(gomock.Any(), "C1").Return(seqErr(errors.New("iter fail")), nil)
			},
			wantIsError: true,
			wantText:    "iter fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleGetMessages(t.Context(), toolReq(tt.args))
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleGetMessages) })
	t.Run("iterator lifetime", func(t *testing.T) { checkIteratorRead(t, false) })

}

// ─── handleGetThread ──────────────────────────────────────────────────────────

func TestServer_handleGetThread(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string
	}{
		{
			name:        "missing channel_id returns error result",
			args:        nil,
			setup:       func(m *mock_source.MockSourceResumeCloser) {},
			wantIsError: true,
			wantText:    "channel_id",
		},
		{
			name:        "missing thread_ts returns error result",
			args:        map[string]any{"channel_id": "C1"},
			setup:       func(m *mock_source.MockSourceResumeCloser) {},
			wantIsError: true,
			wantText:    "thread_ts",
		},
		{
			name: "returns thread messages as JSON",
			args: map[string]any{"channel_id": "C1", "thread_ts": "1000.000001"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllThreadMessages(gomock.Any(), "C1", "1000.000001").Return(seqOf(
					slack.Message{Msg: slack.Msg{Timestamp: "1000.000001", Text: "parent", User: "U1"}},
					slack.Message{Msg: slack.Msg{Timestamp: "1000.000002", Text: "reply", User: "U2"}},
				), nil)
			},
			wantText: "parent",
		},
		{
			name: "ErrNotFound returns informational text",
			args: map[string]any{"channel_id": "C1", "thread_ts": "9999.000001"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllThreadMessages(gomock.Any(), "C1", "9999.000001").Return(nil, source.ErrNotFound)
			},
			wantText: "9999.000001",
		},
		{
			name: "ErrNotSupported returns informational text",
			args: map[string]any{"channel_id": "C1", "thread_ts": "1000.000001"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllThreadMessages(gomock.Any(), "C1", "1000.000001").Return(nil, source.ErrNotSupported)
			},
			wantText: "not support",
		},
		{
			name: "iterator error returns error result",
			args: map[string]any{"channel_id": "C1", "thread_ts": "1000.000001"},
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().AllThreadMessages(gomock.Any(), "C1", "1000.000001").Return(seqErr(errors.New("read fail")), nil)
			},
			wantIsError: true,
			wantText:    "read fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleGetThread(t.Context(), toolReq(tt.args))
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleGetThread) })
	t.Run("iterator lifetime", func(t *testing.T) { checkIteratorRead(t, true) })

}

// ─── handleGetWorkspaceInfo ───────────────────────────────────────────────────

func TestServer_handleGetWorkspaceInfo(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(m *mock_source.MockSourceResumeCloser)
		wantIsError bool
		wantText    string
	}{
		{
			name: "returns workspace info as JSON",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().WorkspaceInfo(gomock.Any()).Return(
					&slack.AuthTestResponse{Team: "Acme Inc", URL: "https://acme.slack.com"},
					nil,
				)
			},
			wantText: "Acme Inc",
		},
		{
			name: "ErrNotFound returns informational text",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotFound)
			},
			wantText: "not available",
		},
		{
			name: "ErrNotSupported returns informational text",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, source.ErrNotSupported)
			},
			wantText: "not available",
		},
		{
			name: "generic error returns error result",
			setup: func(m *mock_source.MockSourceResumeCloser) {
				m.EXPECT().WorkspaceInfo(gomock.Any()).Return(nil, errors.New("db error"))
			},
			wantIsError: true,
			wantText:    "db error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, mock := newTestServer(t, ctrl)
			tt.setup(mock)

			result, err := srv.handleGetWorkspaceInfo(t.Context(), mcplib.CallToolRequest{})
			requireSourceUnlocked(t, srv)
			mock.EXPECT().Close().Return(nil)
			require.NoError(t, srv.loadSource(nil))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("no source takes precedence and releases lock", func(t *testing.T) { checkNilSource(t, (*Server).handleGetWorkspaceInfo) })
	t.Run("replacement waits for source read", func(t *testing.T) {
		checkHandlerRead(t, func(m *mock_source.MockSourceResumeCloser, block func()) {
			m.EXPECT().WorkspaceInfo(gomock.Any()).DoAndReturn(func(context.Context) (*slack.AuthTestResponse, error) { block(); return &slack.AuthTestResponse{}, nil })
		}, (*Server).handleGetWorkspaceInfo, map[string]any{"channel_id": "C1"}, false)
	})

}

// ─── handleLoadSource ─────────────────────────────────────────────────────────

func TestServer_handleLoadSource(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		setupLoader func(ctrl *gomock.Controller) (SourceLoader, *mock_source.MockSourceResumeCloser)
		setupOld    func(ctrl *gomock.Controller) *mock_source.MockSourceResumeCloser
		wantIsError bool
		wantText    string
	}{
		{
			name:        "missing path returns error result",
			args:        nil,
			setupLoader: func(ctrl *gomock.Controller) (SourceLoader, *mock_source.MockSourceResumeCloser) { return nil, nil },
			wantIsError: true,
			wantText:    "path",
		},
		{
			name: "loader error returns error result",
			args: map[string]any{"path": "/bad/path"},
			setupLoader: func(ctrl *gomock.Controller) (SourceLoader, *mock_source.MockSourceResumeCloser) {
				return func(_ context.Context, _ string) (source.SourceResumeCloser, error) {
					return nil, errors.New("no such file")
				}, nil
			},
			wantIsError: true,
			wantText:    "no such file",
		},
		{
			name: "success opens new source and returns info",
			args: map[string]any{"path": "/some/archive.db"},
			setupLoader: func(ctrl *gomock.Controller) (SourceLoader, *mock_source.MockSourceResumeCloser) {
				next := mock_source.NewMockSourceResumeCloser(ctrl)
				next.EXPECT().Name().Return("archive.db").AnyTimes()
				next.EXPECT().Type().Return(source.FDatabase).AnyTimes()
				loader := func(_ context.Context, _ string) (source.SourceResumeCloser, error) {
					return next, nil
				}
				return loader, next
			},
			wantText: "archive.db",
		},
		{
			name: "success opens next source and closes previous source",
			args: map[string]any{"path": "/new/archive.db"},
			setupOld: func(ctrl *gomock.Controller) *mock_source.MockSourceResumeCloser {
				old := mock_source.NewMockSourceResumeCloser(ctrl)
				old.EXPECT().Close().Return(nil).Times(1)
				return old
			},
			setupLoader: func(ctrl *gomock.Controller) (SourceLoader, *mock_source.MockSourceResumeCloser) {
				next := mock_source.NewMockSourceResumeCloser(ctrl)
				next.EXPECT().Name().Return("new-archive.db").AnyTimes()
				next.EXPECT().Type().Return(source.FDatabase).AnyTimes()
				loader := func(_ context.Context, _ string) (source.SourceResumeCloser, error) {
					return next, nil
				}
				return loader, next
			},
			wantText: "new-archive.db",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			srv := New(WithLogger(nil))

			// Optionally pre-install an old source.
			if tt.setupOld != nil {
				srv.src = tt.setupOld(ctrl)
			}

			// Optionally install a custom loader.
			if tt.setupLoader != nil {
				loader, _ := tt.setupLoader(ctrl)
				if loader != nil {
					srv.loader = loader
				}
			}

			result, err := srv.handleLoadSource(t.Context(), toolReq(tt.args))
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
	t.Run("failed loader leaves old source open and readable during loading", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		srv, old := newTestServer(t, ctrl)
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		unblock := func() { once.Do(func() { close(release) }) }
		srv.loader = func(context.Context, string) (source.SourceResumeCloser, error) {
			close(entered)
			<-release
			return nil, assert.AnError
		}
		loadDone := make(chan struct{})
		var loaded *mcplib.CallToolResult
		var loadErr error
		go func() {
			defer close(loadDone)
			loaded, loadErr = srv.handleLoadSource(t.Context(), toolReq(map[string]any{"path": "bad"}))
		}()
		t.Cleanup(func() { unblock(); awaitSignal(t, loadDone) })
		awaitSignal(t, entered)
		old.EXPECT().Channels(gomock.Any()).Return([]slack.Channel{}, nil).Times(2)
		readDone := make(chan struct{})
		var read *mcplib.CallToolResult
		var readErr error
		go func() {
			defer close(readDone)
			read, readErr = srv.handleListChannels(t.Context(), toolReq(nil))
		}()
		t.Cleanup(func() { unblock(); awaitSignal(t, readDone) })
		awaitSignal(t, readDone)
		require.NoError(t, readErr)
		assert.False(t, read.IsError)
		unblock()
		awaitSignal(t, loadDone)
		require.NoError(t, loadErr)
		assert.True(t, loaded.IsError)
		assert.Contains(t, firstText(t, loaded), assert.AnError.Error())
		assert.Same(t, old, srv.src)
		after, err := srv.handleListChannels(t.Context(), toolReq(nil))
		require.NoError(t, err)
		assert.False(t, after.IsError)
		requireSourceUnlocked(t, srv)
		// No Close expectation: neither failure nor either read may close old.
	})
	t.Run("captures metadata before publishing and does not read it afterward", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		old := mock_source.NewMockSourceResumeCloser(ctrl)
		next := mock_source.NewMockSourceResumeCloser(ctrl)
		srv := New(WithSourceLoader(func(context.Context, string) (source.SourceResumeCloser, error) { return next, nil }))
		srv.src = old
		gomock.InOrder(
			next.EXPECT().Name().Return("next"),
			next.EXPECT().Type().Return(source.FDatabase),
			old.EXPECT().Close().Return(nil),
		)
		result, err := srv.handleLoadSource(t.Context(), toolReq(map[string]any{"path": "next"}))
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Equal(t, "Archive \"next\" (type: database) loaded successfully. You can now use the data tools.", firstText(t, result))
		assert.Same(t, next, srv.src)
		requireSourceUnlocked(t, srv)
	})

}

// checkHandlerRead blocks a source method or iterator while a real replacement
// runs. Checking the lock before starting the writer prevents scheduling from
// hiding an unprotected read.
func checkHandlerRead(t *testing.T,
	setup func(*mock_source.MockSourceResumeCloser, func()),
	handler func(*Server, context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error),
	args map[string]any, wantError bool,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	srv, old := newTestServer(t, ctrl)
	next := mock_source.NewMockSourceResumeCloser(ctrl)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var active atomic.Bool
	setup(old, func() {
		active.Store(true)
		close(entered)
		<-release
		active.Store(false)
	})
	old.EXPECT().Close().DoAndReturn(func() error {
		assert.False(t, active.Load(), "closed during source use or iterator cleanup")
		return nil
	})
	readDone := make(chan struct{})
	var result *mcplib.CallToolResult
	var readErr error
	go func() {
		defer close(readDone)
		result, readErr = handler(srv, t.Context(), toolReq(args))
	}()
	t.Cleanup(func() { unblock(); awaitSignal(t, readDone) })
	awaitSignal(t, entered)
	requireSourceLocked(t, srv)
	writeDone, writeStarted := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(writeDone)
		close(writeStarted)
		assert.NoError(t, srv.loadSource(next))
	}()
	t.Cleanup(func() { unblock(); awaitSignal(t, writeDone) })
	awaitSignal(t, writeStarted)
	unblock()
	awaitSignal(t, readDone)
	awaitSignal(t, writeDone)
	require.NoError(t, readErr)
	require.NotNil(t, result)
	assert.Equal(t, wantError, result.IsError)
	assert.Same(t, next, srv.src)
	requireSourceUnlocked(t, srv)
}

func checkNilSource(t *testing.T, handler func(*Server, context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)) {
	t.Helper()
	srv := New()
	result, err := handler(srv, t.Context(), toolReq(nil))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Equal(t, errNoSource.Error(), firstText(t, result))
	requireSourceUnlocked(t, srv)
	next := mock_source.NewMockSourceResumeCloser(gomock.NewController(t))
	require.NoError(t, srv.loadSource(next))
	assert.Same(t, next, srv.src)
}

// checkIteratorRead exercises the lazy sequence after its factory returned,
// including cleanup while yield has returned false on break or handler return.
func checkIteratorRead(t *testing.T, thread bool) {
	t.Helper()
	for _, outcome := range []string{"exhaustion", "iterator error", "cleanup after error", "cleanup after exhaustion", "source error"} {
		t.Run(outcome, func(t *testing.T) {
			checkMessageRead(t, thread, outcome)
		})
	}
	if !thread {
		t.Run("cleanup after limit break", func(t *testing.T) { checkMessageRead(t, false, "cleanup after limit break") })
	}
}

func checkMessageRead(t *testing.T, thread bool, outcome string) {
	t.Helper()
	args := map[string]any{"channel_id": "C1", "thread_ts": "1.0"}
	if outcome == "cleanup after limit break" {
		args["limit"] = float64(1)
	}
	handler := (*Server).handleGetMessages
	if thread {
		handler = (*Server).handleGetThread
	}
	wantError := outcome == "source error" || outcome == "iterator error" || outcome == "cleanup after error"
	checkHandlerRead(t, func(m *mock_source.MockSourceResumeCloser, block func()) {
		factory := func() (iter.Seq2[slack.Message, error], error) {
			if outcome == "source error" {
				block()
				return nil, assert.AnError
			}
			return func(yield func(slack.Message, error) bool) {
				if strings.HasPrefix(outcome, "cleanup") {
					defer block()
				} else {
					block()
				}
				if outcome == "iterator error" || outcome == "cleanup after error" {
					if !yield(slack.Message{Msg: slack.Msg{Timestamp: "1.0", Text: "first"}}, nil) {
						return
					}
					assert.False(t, yield(slack.Message{}, assert.AnError), "iterator error must stop iteration")
					return
				}
				more := yield(slack.Message{Msg: slack.Msg{Timestamp: "1.0", Text: "first"}}, nil)
				if outcome == "cleanup after limit break" {
					assert.False(t, more, "limit must stop iteration")
				}
			}, nil
		}
		if thread {
			m.EXPECT().AllThreadMessages(gomock.Any(), "C1", "1.0").DoAndReturn(func(context.Context, string, string) (iter.Seq2[slack.Message, error], error) { return factory() })
		} else {
			m.EXPECT().AllMessages(gomock.Any(), "C1").DoAndReturn(func(context.Context, string) (iter.Seq2[slack.Message, error], error) { return factory() })
		}
	}, handler, args, wantError)
}
