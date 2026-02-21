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
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v4/source"
	"github.com/rusq/slackdump/v4/source/mock_source"
)

// newTestServer creates a *Server backed by a MockSourcer with minimum
// Name/Type expectations set.
func newTestServer(t *testing.T, ctrl *gomock.Controller) (*Server, *mock_source.MockSourcer) {
	t.Helper()
	m := mock_source.NewMockSourcer(ctrl)
	m.EXPECT().Name().Return("test-archive").AnyTimes()
	m.EXPECT().Type().Return(source.FDatabase).AnyTimes()
	srv := New(m, nil)
	require.NotNil(t, srv)
	return srv, m
}

// toolReq builds a CallToolRequest with the given argument map.
func toolReq(args map[string]any) mcplib.CallToolRequest {
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = args
	return req
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
	ctrl := gomock.NewController(t)
	m := mock_source.NewMockSourcer(ctrl)
	m.EXPECT().Name().Return("x").AnyTimes()
	m.EXPECT().Type().Return(source.FDatabase).AnyTimes()
	// Must not panic when logger is nil.
	assert.NotPanics(t, func() {
		srv := New(m, nil)
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

func TestInstructions(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mock_source.NewMockSourcer(ctrl)
	m.EXPECT().Name().Return("my-archive.db").AnyTimes()
	m.EXPECT().Type().Return(source.FDatabase).AnyTimes()

	got := instructions(m)
	assert.Contains(t, got, "my-archive.db")
	assert.Contains(t, got, "database")
}

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
