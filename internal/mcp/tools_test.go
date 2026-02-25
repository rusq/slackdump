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

func TestHandleListChannels(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleGetChannel ─────────────────────────────────────────────────────────

func TestHandleGetChannel(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleListUsers ──────────────────────────────────────────────────────────

func TestHandleListUsers(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleGetMessages ────────────────────────────────────────────────────────

func TestHandleGetMessages(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleGetThread ──────────────────────────────────────────────────────────

func TestHandleGetThread(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleGetWorkspaceInfo ───────────────────────────────────────────────────

func TestHandleGetWorkspaceInfo(t *testing.T) {
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
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantIsError, isErrorResult(result))
			if tt.wantText != "" {
				assert.Contains(t, firstText(t, result), tt.wantText)
			}
		})
	}
}

// ─── handleLoadSource ─────────────────────────────────────────────────────────

func TestHandleLoadSource(t *testing.T) {
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
			name: "success closes previous source before opening new one",
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
}

// ─── nil source guard ─────────────────────────────────────────────────────────

func TestHandlers_nilSource(t *testing.T) {
	// Every data tool handler must return an error result (not panic) when no
	// source has been loaded yet.
	srv := New(WithLogger(nil)) // no WithSource → srv.src is nil

	req := mcplib.CallToolRequest{}
	ctx := t.Context()

	handlers := []struct {
		name string
		fn   func(context.Context, mcplib.CallToolRequest) (*mcplib.CallToolResult, error)
	}{
		{"list_channels", srv.handleListChannels},
		{"get_channel", srv.handleGetChannel},
		{"list_users", srv.handleListUsers},
		{"get_messages", srv.handleGetMessages},
		{"get_thread", srv.handleGetThread},
		{"get_workspace_info", srv.handleGetWorkspaceInfo},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			result, err := h.fn(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, isErrorResult(result), "expected IsError=true for nil source")
			assert.Contains(t, firstText(t, result), "load_source")
		})
	}
}
