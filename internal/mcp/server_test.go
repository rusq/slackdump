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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

// requireSourceLocked checks the lifetime invariant at a barrier, before a
// writer is started. Undo an unexpected acquisition before reporting failure.
func requireSourceLocked(t *testing.T, srv *Server) {
	t.Helper()
	locked := srv.mu.TryLock()
	if locked {
		srv.mu.Unlock()
	}
	require.False(t, locked, "source read must hold shared access")
}

func requireSourceUnlocked(t *testing.T, srv *Server) {
	t.Helper()
	locked := srv.mu.TryLock()
	if locked {
		srv.mu.Unlock()
	}
	require.True(t, locked, "source access leaked a lock")
}

// awaitSignal uses a timeout only as a deadlock guard, never to order work.
func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for test goroutine")
	}
}

func TestServer_withSource(t *testing.T) {
	t.Run("no source skips callback and releases lock", func(t *testing.T) {
		srv := New()
		result, err := srv.withSource(func(source.Sourcer) (*mcplib.CallToolResult, error) {
			t.Error("callback invoked without a source")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Equal(t, errNoSource.Error(), firstText(t, result))
		requireSourceUnlocked(t, srv)
	})
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(outcome+" releases lock", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, src := newTestServer(t, ctrl)
			want := resultText("done")
			callback := func(got source.Sourcer) (*mcplib.CallToolResult, error) {
				assert.Same(t, src, got)
				requireSourceLocked(t, srv)
				switch outcome {
				case "error":
					return nil, assert.AnError
				case "panic":
					panic("callback panic")
				}
				return want, nil
			}
			if outcome == "panic" {
				assert.PanicsWithValue(t, "callback panic", func() { _, _ = srv.withSource(callback) })
			} else {
				got, err := srv.withSource(callback)
				if outcome == "error" {
					assert.ErrorIs(t, err, assert.AnError)
					assert.Nil(t, got)
				} else {
					require.NoError(t, err)
					assert.Same(t, want, got)
				}
			}
			requireSourceUnlocked(t, srv)
		})
	}
}

func TestServer_loadSource(t *testing.T) {
	for _, tc := range []struct {
		name     string
		old      bool
		closeErr error
	}{
		{name: "first installation"},
		{name: "closes old source", old: true},
		{name: "close error logs and still swaps", old: true, closeErr: errors.New("close failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			var logs bytes.Buffer
			srv := New(WithLogger(slog.New(slog.NewTextHandler(&logs, nil))))
			if tc.old {
				old := mock_source.NewMockSourceResumeCloser(ctrl)
				old.EXPECT().Close().Return(tc.closeErr)
				srv.src = old
			}
			next := mock_source.NewMockSourceResumeCloser(ctrl)
			require.NoError(t, srv.loadSource(next))
			assert.Same(t, next, srv.src)
			requireSourceUnlocked(t, srv)
			if tc.closeErr != nil {
				assert.Contains(t, logs.String(), "error closing previous source")
				assert.Contains(t, logs.String(), "close failed")
			}
		})
	}
	for _, readers := range []int{1, 2} {
		t.Run(fmt.Sprintf("waits for all %d concurrent readers", readers), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, old := newTestServer(t, ctrl)
			next := mock_source.NewMockSourceResumeCloser(ctrl)
			var active atomic.Int32
			closed := make(chan struct{})
			old.EXPECT().Close().DoAndReturn(func() error {
				assert.Zero(t, active.Load(), "closed source with active readers")
				close(closed)
				return nil
			})
			entered := make(chan struct{}, readers)
			releases := make([]chan struct{}, readers)
			done := make([]chan struct{}, readers)
			for i := range readers {
				releases[i], done[i] = make(chan struct{}), make(chan struct{})
			}
			writerDone := make(chan struct{})
			var releaseOnce sync.Once
			releaseAll := func() {
				releaseOnce.Do(func() {
					for _, ch := range releases {
						close(ch)
					}
				})
			}
			// Join readers even when a lock assertion fails before starting the writer.
			t.Cleanup(func() {
				releaseAll()
				for _, ch := range done {
					awaitSignal(t, ch)
				}
			})
			for i := range readers {
				go func() {
					defer close(done[i])
					_, _ = srv.withSource(func(src source.Sourcer) (*mcplib.CallToolResult, error) {
						active.Add(1)
						entered <- struct{}{}
						<-releases[i]
						active.Add(-1)
						return resultText("read"), nil
					})
				}()
			}
			for range readers {
				awaitSignal(t, entered)
			}
			requireSourceLocked(t, srv)
			writerStarted := make(chan struct{})
			go func() {
				defer close(writerDone)
				close(writerStarted)
				assert.NoError(t, srv.loadSource(next))
			}()
			t.Cleanup(func() { releaseAll(); awaitSignal(t, writerDone) })
			awaitSignal(t, writerStarted)
			if readers == 2 {
				// Release one reader without opening the other reader's gate.
				releases[0] <- struct{}{}
				awaitSignal(t, done[0])
				assert.EqualValues(t, 1, active.Load())
				select {
				case <-closed:
					t.Error("closed before remaining reader finished")
				default:
				}
			}
			releaseAll()
			awaitSignal(t, writerDone)
			awaitSignal(t, closed)
			for _, ch := range done {
				awaitSignal(t, ch)
			}
			assert.Same(t, next, srv.src)
			next.EXPECT().Name().Return("replacement")
			result, err := srv.withSource(func(src source.Sourcer) (*mcplib.CallToolResult, error) { return resultText(src.Name()), nil })
			require.NoError(t, err)
			assert.Equal(t, "replacement", firstText(t, result))
			requireSourceUnlocked(t, srv)
		})
	}
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
