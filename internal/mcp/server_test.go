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
	"runtime"
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

func Test_instructions(t *testing.T) {
	got := instructions()
	for _, tool := range []string{"load_source", "list_channels", "get_channel", "list_users", "get_messages", "get_thread", "get_workspace_info", "command_help"} {
		assert.Contains(t, got, tool)
	}
	assert.Contains(t, got, "currently loaded archive")
	assert.NotContains(t, got, "No archive has been loaded yet")
}

// requireSourceAccess checks exclusive availability at a barrier, releasing an
// unexpected acquisition before reporting failure.
func requireSourceAccess(t *testing.T, srv *Server, available bool) {
	t.Helper()
	acquired := srv.access.TryAcquire(sourceAccessCapacity)
	if acquired {
		srv.access.Release(sourceAccessCapacity)
	}
	require.Equal(t, available, acquired, "exclusive source access availability")
}

// awaitWriterQueued establishes that an exclusive waiter reached the gate.
// Call only with a reader held: a one-token probe otherwise succeeds, so a
// failed probe proves the writer is queued, independently of goroutine scheduling.
func awaitWriterQueued(t *testing.T, srv *Server) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for srv.access.TryAcquire(1) {
		srv.access.Release(1)
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for exclusive source waiter")
		default:
			runtime.Gosched()
		}
	}
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
		result, err := srv.withSource(t.Context(), func(source.Sourcer) (*mcplib.CallToolResult, error) {
			t.Error("callback invoked without a source")
			return nil, nil
		})
		require.NoError(t, err)
		assert.Equal(t, errNoSource.Error(), firstText(t, result))
		requireSourceAccess(t, srv, true)
	})
	for _, outcome := range []string{"success", "error", "panic"} {
		t.Run(outcome+" releases lock", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv, src := newTestServer(t, ctrl)
			want := resultText("done")
			callback := func(got source.Sourcer) (*mcplib.CallToolResult, error) {
				assert.Same(t, src, got)
				requireSourceAccess(t, srv, false)
				switch outcome {
				case "error":
					return nil, assert.AnError
				case "panic":
					panic("callback panic")
				}
				return want, nil
			}
			if outcome == "panic" {
				assert.PanicsWithValue(t, "callback panic", func() { _, _ = srv.withSource(t.Context(), callback) })
			} else {
				got, err := srv.withSource(t.Context(), callback)
				if outcome == "error" {
					assert.ErrorIs(t, err, assert.AnError)
					assert.Nil(t, got)
				} else {
					require.NoError(t, err)
					assert.Same(t, want, got)
				}
			}
			requireSourceAccess(t, srv, true)
		})
	}
	t.Run("canceled read returns while exclusive access remains held", func(t *testing.T) {
		srv := New()
		require.NoError(t, srv.access.Acquire(t.Context(), sourceAccessCapacity))
		defer srv.access.Release(sourceAccessCapacity)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		var result *mcplib.CallToolResult
		var err error
		go func() {
			defer close(done)
			result, err = srv.withSource(ctx, func(source.Sourcer) (*mcplib.CallToolResult, error) {
				t.Error("canceled read invoked callback")
				return nil, nil
			})
		}()
		t.Cleanup(func() { cancel(); awaitSignal(t, done) })
		cancel()
		awaitSignal(t, done)
		require.NoError(t, err)
		assert.Equal(t, context.Canceled.Error(), firstText(t, result))
	})

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
			require.NoError(t, srv.loadSource(t.Context(), next))
			assert.Same(t, next, srv.src)
			requireSourceAccess(t, srv, true)
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
					_, _ = srv.withSource(t.Context(), func(src source.Sourcer) (*mcplib.CallToolResult, error) {
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
			requireSourceAccess(t, srv, false)
			go func() {
				defer close(writerDone)
				assert.NoError(t, srv.loadSource(t.Context(), next))
			}()
			t.Cleanup(func() { releaseAll(); awaitSignal(t, writerDone) })
			awaitWriterQueued(t, srv)
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
			result, err := srv.withSource(t.Context(), func(src source.Sourcer) (*mcplib.CallToolResult, error) { return resultText(src.Name()), nil })
			require.NoError(t, err)
			assert.Equal(t, "replacement", firstText(t, result))
			requireSourceAccess(t, srv, true)
		})
	}
}

func TestServer_Close(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hasSource bool
		closeErr  error
	}{
		{name: "without source"},
		{name: "closes once", hasSource: true},
		{name: "close failure still clears ownership", hasSource: true, closeErr: assert.AnError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			srv := New()
			if tc.hasSource {
				src := mock_source.NewMockSourceResumeCloser(ctrl)
				src.EXPECT().Close().Return(tc.closeErr).Times(1)
				srv.src = src
			}
			assert.ErrorIs(t, srv.Close(), tc.closeErr)
			assert.Nil(t, srv.src)
			assert.True(t, srv.closed)
			require.NoError(t, srv.Close())
			result, err := srv.withSource(t.Context(), func(source.Sourcer) (*mcplib.CallToolResult, error) {
				t.Error("read invoked after Close")
				return nil, nil
			})
			require.NoError(t, err)
			assert.Equal(t, errServerClosed.Error(), firstText(t, result))
			next := mock_source.NewMockSourceResumeCloser(ctrl)
			assert.ErrorIs(t, srv.loadSource(t.Context(), next), errServerClosed)
			// Failed publication leaves the caller responsible for next.
			next.EXPECT().Close().Return(nil)
			require.NoError(t, next.Close())
			requireSourceAccess(t, srv, true)
		})
	}
	t.Run("closes replacement without closing initial source twice", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		srv, old := newTestServer(t, ctrl)
		next := mock_source.NewMockSourceResumeCloser(ctrl)
		gomock.InOrder(old.EXPECT().Close().Return(nil), next.EXPECT().Close().Return(nil))
		require.NoError(t, srv.loadSource(t.Context(), next))
		require.NoError(t, srv.Close())
		require.NoError(t, srv.Close())
	})
	t.Run("waits for active reader and concurrent closes run once", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		srv, src := newTestServer(t, ctrl)
		require.NoError(t, srv.access.Acquire(t.Context(), 1))
		var once sync.Once
		release := func() { once.Do(func() { srv.access.Release(1) }) }
		closed := make(chan struct{})
		src.EXPECT().Close().DoAndReturn(func() error { close(closed); return nil }).Times(1)
		done := []chan struct{}{make(chan struct{}), make(chan struct{})}
		for _, ch := range done {
			go func() { defer close(ch); assert.NoError(t, srv.Close()) }()
		}
		t.Cleanup(func() {
			release()
			for _, ch := range done {
				awaitSignal(t, ch)
			}
		})
		awaitWriterQueued(t, srv)
		select {
		case <-closed:
			t.Fatal("closed source with an active reader")
		default:
		}
		release()
		for _, ch := range done {
			awaitSignal(t, ch)
		}
		awaitSignal(t, closed)
		assert.Nil(t, srv.src)
		requireSourceAccess(t, srv, true)
	})
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
