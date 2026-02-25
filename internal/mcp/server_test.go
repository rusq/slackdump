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
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

// newTestServer creates a *Server backed by a MockSourceResumeCloser with
// minimum Name/Type expectations set, pre-loaded via direct field injection.
// It returns the mock typed as *mock_source.MockSourcer for convenience in
// tool handler tests (MockSourceResumeCloser embeds all Sourcer methods).
func newTestServer(t *testing.T, ctrl *gomock.Controller) (*Server, *mock_source.MockSourceResumeCloser) {
	t.Helper()
	m := mock_source.NewMockSourceResumeCloser(ctrl)
	m.EXPECT().Name().Return("test-archive").AnyTimes()
	m.EXPECT().Type().Return(source.FDatabase).AnyTimes()
	srv := New(WithLogger(nil))
	srv.src = m
	require.NotNil(t, srv)
	return srv, m
}

// toolReq builds a CallToolRequest with the given argument map.
func toolReq(args map[string]any) mcplib.CallToolRequest {
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// ─── New / options ────────────────────────────────────────────────────────────

func TestNew_noOptions(t *testing.T) {
	srv := New()
	require.NotNil(t, srv)
	assert.NotNil(t, srv.mcp)
	assert.Nil(t, srv.src) // no source by default
	assert.NotNil(t, srv.logger)
	assert.NotNil(t, srv.loader)
}

func TestNew_withLogger_nil(t *testing.T) {
	// A nil logger must not panic and must fall back to slog.Default().
	assert.NotPanics(t, func() {
		srv := New(WithLogger(nil))
		assert.NotNil(t, srv.logger)
	})
}

func TestNew_notNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, _ := newTestServer(t, ctrl)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.mcp)
	assert.NotNil(t, srv.src)
	assert.NotNil(t, srv.logger)
}

func TestNew_nilLogger(t *testing.T) {
	// Must not panic when logger option is nil.
	assert.NotPanics(t, func() {
		srv := New(WithLogger(nil))
		assert.NotNil(t, srv.logger)
	})
}

func TestAddTool(t *testing.T) {
	ctrl := gomock.NewController(t)
	srv, _ := newTestServer(t, ctrl)

	extra := mcpsrv.ServerTool{
		Tool: mcplib.NewTool("extra_tool", mcplib.WithDescription("extra")),
		Handler: func(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			return resultText("ok"), nil
		},
	}
	assert.NotPanics(t, func() {
		srv.AddTool(extra)
	})
}

func TestInstructions_withSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mock_source.NewMockSourcer(ctrl)
	m.EXPECT().Name().Return("my-archive.db").AnyTimes()
	m.EXPECT().Type().Return(source.FDatabase).AnyTimes()

	got := instructions(m)
	assert.Contains(t, got, "my-archive.db")
	assert.Contains(t, got, "database")
}

func TestInstructions_nilSource(t *testing.T) {
	got := instructions(nil)
	assert.Contains(t, got, "load_source")
	assert.NotContains(t, got, "nil")
}

// ─── source helper methods ────────────────────────────────────────────────────

func TestLoadSource_closesOld(t *testing.T) {
	ctrl := gomock.NewController(t)

	// Old source: must be closed.
	old := mock_source.NewMockSourceResumeCloser(ctrl)
	old.EXPECT().Close().Return(nil).Times(1)

	// New source.
	next := mock_source.NewMockSourceResumeCloser(ctrl)

	srv := New()
	srv.src = old

	err := srv.loadSource(next)
	require.NoError(t, err)
	assert.Equal(t, next, srv.src)
}

func TestLoadSource_closeError_stillSwaps(t *testing.T) {
	ctrl := gomock.NewController(t)

	old := mock_source.NewMockSourceResumeCloser(ctrl)
	old.EXPECT().Close().Return(errors.New("close failed")).Times(1)

	next := mock_source.NewMockSourceResumeCloser(ctrl)

	srv := New()
	srv.src = old

	// Even when Close() errors, loadSource must swap the source.
	err := srv.loadSource(next)
	require.NoError(t, err)
	assert.Equal(t, next, srv.src)
}

// ─── result helpers ───────────────────────────────────────────────────────────

func TestResultText(t *testing.T) {
	r := resultText("hello")
	require.NotNil(t, r)
	assert.False(t, r.IsError)
	require.Len(t, r.Content, 1)
	txt, ok := r.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	assert.Equal(t, "hello", txt.Text)
}

func TestResultErr(t *testing.T) {
	r := resultErr(assert.AnError)
	require.NotNil(t, r)
	assert.True(t, r.IsError)
	require.Len(t, r.Content, 1)
	txt, ok := r.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	assert.Equal(t, assert.AnError.Error(), txt.Text)
}

func TestResultJSON(t *testing.T) {
	type payload struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	r, err := resultJSON(payload{ID: "C1", Name: "general"})
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.False(t, r.IsError)
	require.Len(t, r.Content, 1)
	txt, ok := r.Content[0].(mcplib.TextContent)
	require.True(t, ok)
	assert.Contains(t, txt.Text, "C1")
	assert.Contains(t, txt.Text, "general")
}

// ─── argument helpers ─────────────────────────────────────────────────────────

func TestStringArg(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		argName string
		wantVal string
		wantOK  bool
	}{
		{
			name:    "present string",
			args:    map[string]any{"key": "value"},
			argName: "key",
			wantVal: "value",
			wantOK:  true,
		},
		{
			name:    "missing key",
			args:    map[string]any{},
			argName: "key",
			wantVal: "",
			wantOK:  false,
		},
		{
			name:    "wrong type",
			args:    map[string]any{"key": 42},
			argName: "key",
			wantVal: "",
			wantOK:  false,
		},
		{
			name:    "nil args",
			args:    nil,
			argName: "key",
			wantVal: "",
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := toolReq(tt.args)
			got, ok := stringArg(req, tt.argName)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVal, got)
		})
	}
}

func TestIntArg(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		argName    string
		defaultVal int
		want       int
	}{
		{
			name:       "float64 value",
			args:       map[string]any{"n": float64(42)},
			argName:    "n",
			defaultVal: 0,
			want:       42,
		},
		{
			name:       "int value",
			args:       map[string]any{"n": 7},
			argName:    "n",
			defaultVal: 0,
			want:       7,
		},
		{
			name:       "missing key uses default",
			args:       map[string]any{},
			argName:    "n",
			defaultVal: 99,
			want:       99,
		},
		{
			name:       "nil args uses default",
			args:       nil,
			argName:    "n",
			defaultVal: 5,
			want:       5,
		},
		{
			name:       "wrong type uses default",
			args:       map[string]any{"n": "not-a-number"},
			argName:    "n",
			defaultVal: 3,
			want:       3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := toolReq(tt.args)
			got := intArg(req, tt.argName, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBoolArg(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		argName    string
		defaultVal bool
		want       bool
	}{
		{
			name:       "true value",
			args:       map[string]any{"flag": true},
			argName:    "flag",
			defaultVal: false,
			want:       true,
		},
		{
			name:       "false value",
			args:       map[string]any{"flag": false},
			argName:    "flag",
			defaultVal: true,
			want:       false,
		},
		{
			name:       "missing key uses default true",
			args:       map[string]any{},
			argName:    "flag",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "nil args uses default",
			args:       nil,
			argName:    "flag",
			defaultVal: true,
			want:       true,
		},
		{
			name:       "wrong type uses default",
			args:       map[string]any{"flag": "yes"},
			argName:    "flag",
			defaultVal: false,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := toolReq(tt.args)
			got := boolArg(req, tt.argName, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}
