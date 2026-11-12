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

// Package presence keeps the Slack user presence "active" by emulating the
// activity signals that the official Slack client sends over its websocket.
//
// The Slack client keeps a websocket connection to the message server
// (the "flannel" gateway) and:
//
//   - sends {"type":"ping","id":N} every 10 seconds, the server replies with
//     {"type":"pong","reply_to":N};
//   - sends {"type":"tickle","reason":R,"id":N} on user activity, where R is
//     one of "focus", "mousedown", "keydown", or "polling" (the desktop app
//     polls the OS idle time and tickles on a timer).  The client throttles
//     tickles to at most one per 19 seconds, and the server does not reply.
//
// An open connection alone does not keep the user active: tested with
// connection_count=1, Slack set auto_away about 30 minutes after the last
// tickle (Slack API docs say 10 minutes).  Sending a tickle every few minutes
// over a live connection keeps the user active.
package presence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rusq/slack"
)

const (
	// MinInterval is the minimum interval between tickles, matching the
	// throttle in the Slack client.
	MinInterval = 19 * time.Second
	// DefaultInterval is the default interval between tickles.  It must be
	// well below the auto-away threshold (documented as 10 minutes,
	// observed at 30 minutes).
	DefaultInterval = 4 * time.Minute
	// DefaultJitter is the default jitter fraction applied to the interval.
	DefaultJitter = 0.25
	// DefaultPing is the default ping interval, same as the Slack client.
	DefaultPing = 10 * time.Second
	// DefaultReason is the tickle reason used by the desktop client when it
	// polls the OS idle time.
	DefaultReason = "polling"

	// FlannelHost is the Slack message server websocket host.
	FlannelHost = "wss-primary.slack.com"

	writeTimeout  = 10 * time.Second
	checkTimeout  = 15 * time.Second
	missedPongs   = 3
	defMinBackoff = time.Second
	defMaxBackoff = time.Minute
	// healthySession is the session duration after which the reconnect
	// backoff is reset.
	healthySession = time.Minute
)

var (
	ErrGoodbye     = errors.New("server said goodbye")
	ErrPongTimeout = errors.New("pong timeout")
	ErrNoUserID    = errors.New("user ID is empty")
	ErrNoURL       = errors.New("websocket URL function is nil")
)

// Config is the Keeper configuration.
type Config struct {
	// Interval is the base interval between tickles.
	Interval time.Duration
	// Jitter is the fraction of Interval to randomly add or subtract, [0, 1).
	Jitter float64
	// Ping is the websocket ping interval.
	Ping time.Duration
	// Reason is the tickle reason.
	Reason string
	// Check enables calling users.getPresence after each tickle.
	Check bool
	// CheckEvery, if positive, calls users.getPresence on its own timer,
	// independently of tickles.
	CheckEvery time.Duration
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Interval: DefaultInterval,
		Jitter:   DefaultJitter,
		Ping:     DefaultPing,
		Reason:   DefaultReason,
	}
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if c.Interval < MinInterval {
		return fmt.Errorf("interval %s is less than the minimum %s", c.Interval, MinInterval)
	}
	if c.Jitter < 0 || c.Jitter >= 1 {
		return fmt.Errorf("jitter %v is out of range [0, 1)", c.Jitter)
	}
	if c.Ping <= 0 {
		return fmt.Errorf("ping interval must be positive, got %s", c.Ping)
	}
	if c.Reason == "" {
		return errors.New("tickle reason is empty")
	}
	if c.CheckEvery < 0 {
		return fmt.Errorf("check interval must not be negative, got %s", c.CheckEvery)
	}
	return nil
}

// URLFunc returns the websocket URL to connect to.  It is called on every
// (re)connection.
type URLFunc func(ctx context.Context) (string, error)

// PresenceGetter is the subset of the Slack API used to verify presence.
type PresenceGetter interface {
	GetUserPresenceContext(ctx context.Context, user string) (*slack.UserPresence, error)
}

// Keeper keeps the user presence active.
type Keeper struct {
	userID string
	url    URLFunc
	header http.Header
	cfg    Config

	dialer *websocket.Dialer
	api    PresenceGetter
	lg     *slog.Logger
	rnd    func() float64

	minBackoff time.Duration
	maxBackoff time.Duration

	id int
}

// Option configures the Keeper.
type Option func(*Keeper)

// WithLogger sets the logger.
func WithLogger(lg *slog.Logger) Option {
	return func(k *Keeper) {
		if lg != nil {
			k.lg = lg
		}
	}
}

// WithDialer sets the websocket dialer.
func WithDialer(d *websocket.Dialer) Option {
	return func(k *Keeper) {
		if d != nil {
			k.dialer = d
		}
	}
}

// WithPresenceGetter sets the API used to verify the presence when
// Config.Check is set.
func WithPresenceGetter(api PresenceGetter) Option {
	return func(k *Keeper) {
		k.api = api
	}
}

// New creates a new Keeper for the user userID.  urlFn returns the websocket
// URL, header contains the headers to send on the websocket handshake (i.e.
// Cookie, User-Agent, Origin).
func New(userID string, urlFn URLFunc, header http.Header, cfg Config, opts ...Option) (*Keeper, error) {
	if userID == "" {
		return nil, ErrNoUserID
	}
	if urlFn == nil {
		return nil, ErrNoURL
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	k := &Keeper{
		userID: userID,
		url:    urlFn,
		header: header.Clone(),
		cfg:    cfg,
		dialer: &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 30 * time.Second,
		},
		lg:         slog.Default(),
		rnd:        rand.Float64,
		minBackoff: defMinBackoff,
		maxBackoff: defMaxBackoff,
	}
	for _, o := range opts {
		o(k)
	}
	return k, nil
}

// FlannelURL returns the message server websocket URL, as used by the Slack
// web client, for the given client token and team ID.
func FlannelURL(token, teamID string) string {
	startArgs := url.Values{
		"agent":              {"client"},
		"org_wide_aware":     {"true"},
		"eac_cache_ts":       {"true"},
		"cache_ts":           {"0"},
		"name_tagging":       {"true"},
		"only_self_subteams": {"true"},
		"connect_only":       {"true"},
		"ms_latest":          {"true"},
	}
	q := url.Values{
		"token":                 {token},
		"sync_desync":           {"1"},
		"slack_client":          {"desktop"},
		"start_args":            {"?" + startArgs.Encode()},
		"no_query_on_subscribe": {"1"},
		"flannel":               {"3"},
		"lazy_channels":         {"1"},
		"gateway_server":        {teamID + "-1"},
		"batch_presence_aware":  {"1"},
	}
	u := url.URL{Scheme: "wss", Host: FlannelHost, Path: "/", RawQuery: q.Encode()}
	return u.String()
}

// Run keeps the presence active until ctx is cancelled, reconnecting with
// exponential backoff on connection failures.  It returns nil when ctx is
// cancelled.
func (k *Keeper) Run(ctx context.Context) error {
	backoff := k.minBackoff
	for {
		start := time.Now()
		err := k.session(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if time.Since(start) > healthySession {
			backoff = k.minBackoff
		}
		k.lg.WarnContext(ctx, "connection lost", "error", err, "retry_in", backoff)
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, k.maxBackoff)
	}
}

// outMsg is an outgoing websocket message.
type outMsg struct {
	Type   string   `json:"type"`
	Reason string   `json:"reason,omitempty"`
	IDs    []string `json:"ids,omitempty"`
	ID     int      `json:"id"`
}

// inEvent is the subset of incoming event fields that the Keeper uses.
type inEvent struct {
	Type     string   `json:"type"`
	User     string   `json:"user,omitempty"`
	Users    []string `json:"users,omitempty"`
	Presence string   `json:"presence,omitempty"`
	ReplyTo  int      `json:"reply_to,omitempty"`
	Error    *struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error,omitempty"`
}

// session runs a single websocket connection until it fails or ctx is
// cancelled.
func (k *Keeper) session(ctx context.Context) error {
	u, err := k.url(ctx)
	if err != nil {
		return fmt.Errorf("get websocket url: %w", err)
	}
	conn, resp, err := k.dialer.DialContext(ctx, u, k.header)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("dial: %w (status %s)", err, resp.Status)
		}
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	k.lg.DebugContext(ctx, "websocket connected")

	done := make(chan struct{})
	defer close(done)
	events := make(chan inEvent, 16)
	errC := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				errC <- err
				return
			}
			var ev inEvent
			if err := json.Unmarshal(data, &ev); err != nil {
				k.lg.Debug("skipping undecodable message", "error", err)
				continue
			}
			select {
			case events <- ev:
			case <-done:
				return
			}
		}
	}()

	pingT := time.NewTicker(k.cfg.Ping)
	defer pingT.Stop()
	// tickle timer is armed on "hello".
	tickleT := time.NewTimer(0)
	if !tickleT.Stop() {
		<-tickleT.C
	}
	defer tickleT.Stop()
	// optional presence check timer, nil channel blocks forever.
	var checkC <-chan time.Time
	if k.cfg.CheckEvery > 0 {
		checkT := time.NewTicker(k.cfg.CheckEvery)
		defer checkT.Stop()
		checkC = checkT.C
	}

	lastPong := time.Now()
	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(time.Second))
			return ctx.Err()
		case err := <-errC:
			return fmt.Errorf("read: %w", err)
		case ev := <-events:
			if err := k.handle(ctx, conn, ev, tickleT, &lastPong); err != nil {
				return err
			}
		case <-pingT.C:
			if time.Since(lastPong) > missedPongs*k.cfg.Ping {
				return ErrPongTimeout
			}
			if err := k.send(conn, outMsg{Type: "ping"}); err != nil {
				return err
			}
		case <-tickleT.C:
			if err := k.tickle(ctx, conn); err != nil {
				return err
			}
			tickleT.Reset(k.nextInterval())
		case <-checkC:
			k.check(ctx)
		}
	}
}

// handle processes an incoming event.
func (k *Keeper) handle(ctx context.Context, conn *websocket.Conn, ev inEvent, tickleT *time.Timer, lastPong *time.Time) error {
	switch ev.Type {
	case "hello":
		k.lg.InfoContext(ctx, "connected to the message server")
		if err := k.send(conn, outMsg{Type: "presence_sub", IDs: []string{k.userID}}); err != nil {
			return err
		}
		if err := k.tickle(ctx, conn); err != nil {
			return err
		}
		tickleT.Reset(k.nextInterval())
	case "pong":
		*lastPong = time.Now()
	case "presence_change":
		if ev.User == k.userID || slices.Contains(ev.Users, k.userID) {
			k.lg.InfoContext(ctx, "presence changed", "presence", ev.Presence)
		}
	case "manual_presence_change":
		if ev.Presence == "away" {
			k.lg.WarnContext(ctx, "presence manually set to away, Slack will show you as away until you change it back", "presence", ev.Presence)
		} else {
			k.lg.InfoContext(ctx, "manual presence changed", "presence", ev.Presence)
		}
	case "goodbye":
		return ErrGoodbye
	case "error":
		if ev.Error != nil {
			k.lg.WarnContext(ctx, "server error", "code", ev.Error.Code, "msg", ev.Error.Msg)
		}
	}
	return nil
}

// tickle sends the activity signal and optionally checks the presence.
func (k *Keeper) tickle(ctx context.Context, conn *websocket.Conn) error {
	if err := k.send(conn, outMsg{Type: "tickle", Reason: k.cfg.Reason}); err != nil {
		return err
	}
	k.lg.DebugContext(ctx, "tickle sent", "reason", k.cfg.Reason)
	if k.cfg.Check {
		k.check(ctx)
	}
	return nil
}

// check calls users.getPresence and logs the result.
func (k *Keeper) check(ctx context.Context) {
	if k.api == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	p, err := k.api.GetUserPresenceContext(ctx, k.userID)
	if err != nil {
		k.lg.WarnContext(ctx, "presence check failed", "error", err)
		return
	}
	k.lg.InfoContext(ctx, "presence",
		"presence", p.Presence,
		"online", p.Online,
		"auto_away", p.AutoAway,
		"manual_away", p.ManualAway,
		"connection_count", p.ConnectionCount,
	)
}

// send writes the message, assigning the next message ID.  It must only be
// called from the session goroutine.
func (k *Keeper) send(conn *websocket.Conn, m outMsg) error {
	k.id++
	m.ID = k.id
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}
	if err := conn.WriteJSON(m); err != nil {
		return fmt.Errorf("send %s: %w", m.Type, err)
	}
	return nil
}

// nextInterval returns the interval until the next tickle with jitter
// applied, never less than MinInterval.
func (k *Keeper) nextInterval() time.Duration {
	f := 1 + k.cfg.Jitter*(2*k.rnd()-1)
	d := time.Duration(float64(k.cfg.Interval) * f)
	return max(d, MinInterval)
}
