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

package presence

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUser = "U123"

var discard = slog.New(slog.DiscardHandler)

// wsServer starts a test websocket server that runs fn for each connection.
func wsServer(t *testing.T, fn func(t *testing.T, r *http.Request, c *websocket.Conn)) (URLFunc, *httptest.Server) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer c.Close()
		fn(t, r, c)
	}))
	t.Cleanup(srv.Close)
	u := "ws" + strings.TrimPrefix(srv.URL, "http")
	return func(context.Context) (string, error) { return u, nil }, srv
}

func testConfig() Config {
	c := DefaultConfig()
	c.Ping = time.Hour
	return c
}

func readMsg(t *testing.T, c *websocket.Conn) outMsg {
	t.Helper()
	var m outMsg
	require.NoError(t, c.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, c.ReadJSON(&m))
	return m
}

type fakeGetter struct {
	calls atomic.Int32
	user  atomic.Value
	// block makes the call hang until the context is cancelled.
	block bool
}

func (f *fakeGetter) GetUserPresenceContext(ctx context.Context, user string) (*slack.UserPresence, error) {
	f.calls.Add(1)
	f.user.Store(user)
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &slack.UserPresence{Presence: "active", Online: true}, nil
}

func TestKeeper_session(t *testing.T) {
	t.Run("hello subscribes and tickles, goodbye ends", func(t *testing.T) {
		got := make(chan []outMsg, 1)
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			assert.Equal(t, "d=xoxd-test", r.Header.Get("Cookie"))
			assert.Equal(t, "https://app.slack.com", r.Header.Get("Origin"))
			var msgs []outMsg
			defer func() { got <- msgs }()
			require.NoError(t, c.WriteJSON(map[string]any{"type": "hello"}))
			msgs = []outMsg{readMsg(t, c), readMsg(t, c)}
			require.NoError(t, c.WriteJSON(map[string]any{"type": "presence_change", "user": testUser, "presence": "active"}))
			require.NoError(t, c.WriteJSON(map[string]any{"type": "goodbye"}))
			_, _, _ = c.ReadMessage() // wait for client to close
		})
		hdr := http.Header{"Cookie": {"d=xoxd-test"}, "Origin": {"https://app.slack.com"}}
		k, err := New(testUser, urlFn, hdr, testConfig(), WithLogger(discard))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorIs(t, err, ErrGoodbye)

		var msgs []outMsg
		select {
		case msgs = <-got:
		case <-time.After(5 * time.Second):
			t.Fatal("server did not report messages")
		}
		require.Len(t, msgs, 2)
		assert.Equal(t, outMsg{Type: "presence_sub", IDs: []string{testUser}, ID: 1}, msgs[0])
		assert.Equal(t, outMsg{Type: "tickle", Reason: "polling", ID: 2}, msgs[1])
	})
	t.Run("ping is answered, then pong timeout", func(t *testing.T) {
		var pings atomic.Int32
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			for {
				var m outMsg
				if err := c.ReadJSON(&m); err != nil {
					return
				}
				// reply to the first ping only
				if m.Type == "ping" && pings.Add(1) == 1 {
					_ = c.WriteJSON(map[string]any{"type": "pong", "reply_to": m.ID})
				}
			}
		})
		cfg := testConfig()
		cfg.Ping = 20 * time.Millisecond
		k, err := New(testUser, urlFn, nil, cfg, WithLogger(discard))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorIs(t, err, ErrPongTimeout)
		assert.GreaterOrEqual(t, pings.Load(), int32(missedPongs))
	})
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			require.NoError(t, c.WriteJSON(map[string]any{"type": "hello"}))
			readMsg(t, c)
			readMsg(t, c)
			cancel()
			_, _, _ = c.ReadMessage()
		})
		k, err := New(testUser, urlFn, nil, testConfig(), WithLogger(discard))
		require.NoError(t, err)

		err = k.session(ctx)
		assert.ErrorIs(t, err, context.Canceled)
	})
	t.Run("check calls users.getPresence", func(t *testing.T) {
		fg := &fakeGetter{}
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			require.NoError(t, c.WriteJSON(map[string]any{"type": "hello"}))
			readMsg(t, c)
			readMsg(t, c)
			assert.Eventually(t, func() bool { return fg.calls.Load() == 1 }, 5*time.Second, 5*time.Millisecond)
			require.NoError(t, c.WriteJSON(map[string]any{"type": "goodbye"}))
			_, _, _ = c.ReadMessage()
		})
		cfg := testConfig()
		cfg.Check = true
		k, err := New(testUser, urlFn, nil, cfg, WithLogger(discard), WithPresenceGetter(fg))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorIs(t, err, ErrGoodbye)
		assert.Equal(t, int32(1), fg.calls.Load())
		assert.Equal(t, testUser, fg.user.Load())
	})
	t.Run("check-every polls independently of tickles", func(t *testing.T) {
		fg := &fakeGetter{}
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			require.NoError(t, c.WriteJSON(map[string]any{"type": "hello"}))
			readMsg(t, c)
			readMsg(t, c)
			assert.Eventually(t, func() bool { return fg.calls.Load() >= 2 }, 5*time.Second, 5*time.Millisecond)
			require.NoError(t, c.WriteJSON(map[string]any{"type": "goodbye"}))
			_, _, _ = c.ReadMessage()
		})
		cfg := testConfig()
		cfg.CheckEvery = 10 * time.Millisecond
		k, err := New(testUser, urlFn, nil, cfg, WithLogger(discard), WithPresenceGetter(fg))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorIs(t, err, ErrGoodbye)
		assert.GreaterOrEqual(t, fg.calls.Load(), int32(2))
	})
	t.Run("slow check does not block pings", func(t *testing.T) {
		urlFn, _ := wsServer(t, func(t *testing.T, r *http.Request, c *websocket.Conn) {
			require.NoError(t, c.WriteJSON(map[string]any{"type": "hello"}))
			deadline := time.Now().Add(300 * time.Millisecond)
			for time.Now().Before(deadline) {
				var m outMsg
				require.NoError(t, c.SetReadDeadline(time.Now().Add(time.Second)))
				if err := c.ReadJSON(&m); err != nil {
					return
				}
				if m.Type == "ping" {
					require.NoError(t, c.WriteJSON(map[string]any{"type": "pong", "reply_to": m.ID}))
				}
			}
			require.NoError(t, c.WriteJSON(map[string]any{"type": "goodbye"}))
			_, _, _ = c.ReadMessage()
		})
		cfg := testConfig()
		cfg.Ping = 20 * time.Millisecond
		cfg.Check = true
		cfg.CheckEvery = 10 * time.Millisecond
		fg := &fakeGetter{block: true}
		k, err := New(testUser, urlFn, nil, cfg, WithLogger(discard), WithPresenceGetter(fg))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorIs(t, err, ErrGoodbye, "a hanging presence check must not cause a pong timeout")
		assert.Equal(t, int32(1), fg.calls.Load(), "only one check should be in flight")
	})
	t.Run("dial error", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		t.Cleanup(srv.Close)
		urlFn := func(context.Context) (string, error) { return "ws" + strings.TrimPrefix(srv.URL, "http"), nil }
		k, err := New(testUser, urlFn, nil, testConfig(), WithLogger(discard))
		require.NoError(t, err)

		err = k.session(t.Context())
		assert.ErrorContains(t, err, "404")
	})
}

func TestKeeper_Run(t *testing.T) {
	t.Run("reconnects until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		var calls atomic.Int32
		urlFn := func(context.Context) (string, error) {
			if calls.Add(1) == 3 {
				cancel()
			}
			return "", errors.New("boom")
		}
		k, err := New(testUser, urlFn, nil, testConfig(), WithLogger(discard))
		require.NoError(t, err)
		k.minBackoff = time.Millisecond
		k.maxBackoff = 2 * time.Millisecond

		assert.NoError(t, k.Run(ctx))
		assert.Equal(t, int32(3), calls.Load())
	})
}

func TestKeeper_nextInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		jitter   float64
		rnd      float64
		want     time.Duration
	}{
		{"no jitter", 4 * time.Minute, 0, 0.9, 4 * time.Minute},
		{"lowest", 4 * time.Minute, 0.25, 0, 3 * time.Minute},
		{"middle", 4 * time.Minute, 0.25, 0.5, 4 * time.Minute},
		{"highest", 4 * time.Minute, 0.25, 1, 5 * time.Minute},
		{"clamped to minimum", MinInterval, 0.5, 0, MinInterval},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Keeper{cfg: Config{Interval: tt.interval, Jitter: tt.jitter}, rnd: func() float64 { return tt.rnd }}
			assert.Equal(t, tt.want, k.nextInterval())
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	mod := func(fn func(*Config)) Config {
		c := DefaultConfig()
		fn(&c)
		return c
	}
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"default", DefaultConfig(), false},
		{"interval too short", mod(func(c *Config) { c.Interval = time.Second }), true},
		{"negative jitter", mod(func(c *Config) { c.Jitter = -0.1 }), true},
		{"jitter one", mod(func(c *Config) { c.Jitter = 1 }), true},
		{"zero ping", mod(func(c *Config) { c.Ping = 0 }), true},
		{"empty reason", mod(func(c *Config) { c.Reason = "" }), true},
		{"negative check-every", mod(func(c *Config) { c.CheckEvery = -time.Second }), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.cfg.Validate() != nil)
		})
	}
}

func TestFlannelURL(t *testing.T) {
	u, err := url.Parse(FlannelURL("xoxc-1", "T1"))
	require.NoError(t, err)
	assert.Equal(t, "wss", u.Scheme)
	assert.Equal(t, FlannelHost, u.Host)
	q := u.Query()
	assert.Equal(t, "xoxc-1", q.Get("token"))
	assert.Equal(t, "T1-1", q.Get("gateway_server"))
	assert.Equal(t, "desktop", q.Get("slack_client"))
	assert.Equal(t, "3", q.Get("flannel"))
	sa, err := url.ParseQuery(strings.TrimPrefix(q.Get("start_args"), "?"))
	require.NoError(t, err)
	assert.Equal(t, "client", sa.Get("agent"))
	assert.Equal(t, "true", sa.Get("connect_only"))
}

func TestNew(t *testing.T) {
	urlFn := func(context.Context) (string, error) { return "", nil }
	t.Run("no user", func(t *testing.T) {
		_, err := New("", urlFn, nil, DefaultConfig())
		assert.ErrorIs(t, err, ErrNoUserID)
	})
	t.Run("no url", func(t *testing.T) {
		_, err := New(testUser, nil, nil, DefaultConfig())
		assert.ErrorIs(t, err, ErrNoURL)
	})
	t.Run("invalid config", func(t *testing.T) {
		_, err := New(testUser, urlFn, nil, Config{})
		assert.Error(t, err)
	})
}
