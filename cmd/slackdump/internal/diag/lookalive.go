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

package diag

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackauth"

	"github.com/rusq/slackdump/v4/auth"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/presence"
)

var cmdLookAlive = &base.Command{
	UsageLine:   "slackdump tools lookalive [flags]",
	Short:       "keep your Slack presence active (green)",
	RequireAuth: true,
	HideWizard:  true,
	PrintFlags:  true,
	FlagMask:    cfg.OmitAll &^ cfg.OmitAuthFlags &^ cfg.OmitWorkspaceFlag,
	Long: `
# Look Alive tool

Keeps your Slack presence active (green) while it runs, like a mouse jiggler,
but without touching the mouse.

It connects to the Slack message server the same way the Slack client does,
and periodically sends the "tickle" activity signal that the desktop app sends
when you are using your computer.  An open connection alone is not enough:
Slack marks you as away after about 30 minutes without activity signals (the
Slack API documentation says 10), so the default interval is 4 minutes ± 25%.

Use -check-every to watch your presence on a separate timer.

Use -check to call users.getPresence after each tickle and log what Slack
reports.  Presence changes for your user are always logged.

If you have manually set yourself as away, Slack will keep showing you as
away; use -set-auto to switch your presence back to automatic on start.

Press Ctrl+C to stop, or use -duration to stop automatically.

NOTE: this shows up as one more connected Slack client, and your workspace
policies may not allow it.
`,
}

var lookAliveParams = struct {
	cfg      presence.Config
	duration time.Duration
	setAuto  bool
	rtm      bool
}{
	cfg: presence.DefaultConfig(),
}

func init() {
	cmdLookAlive.Run = runLookAlive
	fs := &cmdLookAlive.Flag
	fs.DurationVar(&lookAliveParams.cfg.Interval, "interval", presence.DefaultInterval, fmt.Sprintf("base interval between activity signals (minimum %s)", presence.MinInterval))
	fs.Float64Var(&lookAliveParams.cfg.Jitter, "jitter", presence.DefaultJitter, "random jitter fraction applied to the interval, [0, 1)")
	fs.DurationVar(&lookAliveParams.cfg.Ping, "ping", presence.DefaultPing, "websocket ping interval")
	fs.StringVar(&lookAliveParams.cfg.Reason, "reason", presence.DefaultReason, "activity signal reason (polling, focus, mousedown, keydown)")
	fs.BoolVar(&lookAliveParams.cfg.Check, "check", false, "check presence with users.getPresence after each activity signal")
	fs.DurationVar(&lookAliveParams.cfg.CheckEvery, "check-every", 0, "check presence with users.getPresence at this interval, independently of activity signals (0 disables)")
	fs.DurationVar(&lookAliveParams.duration, "duration", 0, "stop after this duration (0 runs until interrupted)")
	fs.BoolVar(&lookAliveParams.setAuto, "set-auto", false, "set presence to automatic on start, if it was manually set to away")
	fs.BoolVar(&lookAliveParams.rtm, "rtm", false, "use rtm.connect to get the websocket URL instead of the client message server URL")
}

func runLookAlive(ctx context.Context, cmd *base.Command, args []string) error {
	lg := cfg.Log
	p := lookAliveParams

	if p.duration < 0 {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("duration must not be negative, got %s", p.duration)
	}
	if err := p.cfg.Validate(); err != nil {
		base.SetExitStatus(base.SInvalidParameters)
		return err
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	info, err := prov.Test(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}
	hcl, err := prov.HTTPClient()
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	api := slack.New(prov.SlackToken(), slack.OptionHTTPClient(hcl))

	if err := ensureAutoPresence(ctx, api, info.UserID, p.setAuto); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	urlFn := func(context.Context) (string, error) {
		return presence.FlannelURL(prov.SlackToken(), info.TeamID), nil
	}
	if p.rtm {
		urlFn = func(ctx context.Context) (string, error) {
			_, u, err := api.ConnectRTMContext(ctx)
			return u, err
		}
	}

	k, err := presence.New(info.UserID, urlFn, wsHeader(prov.Cookies()), p.cfg,
		presence.WithLogger(lg),
		presence.WithPresenceGetter(api),
	)
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	if p.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.duration)
		defer cancel()
	}
	lg.InfoContext(ctx, "keeping presence active, press Ctrl+C to stop",
		"user", info.UserID, "team", info.Team, "interval", p.cfg.Interval, "jitter", p.cfg.Jitter)
	if err := k.Run(ctx); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	lg.InfoContext(ctx, "stopped")
	return nil
}

type presenceAPI interface {
	GetUserPresenceContext(ctx context.Context, user string) (*slack.UserPresence, error)
	SetUserPresenceContext(ctx context.Context, presence string) error
}

// ensureAutoPresence checks if the user has manually set their presence to
// away, and if setAuto is true, switches it back to automatic.  Otherwise it
// logs a warning.
func ensureAutoPresence(ctx context.Context, api presenceAPI, userID string, setAuto bool) error {
	up, err := api.GetUserPresenceContext(ctx, userID)
	if err != nil {
		return fmt.Errorf("get presence: %w", err)
	}
	cfg.Log.InfoContext(ctx, "current presence", "presence", up.Presence, "manual_away", up.ManualAway, "connection_count", up.ConnectionCount)
	if !up.ManualAway {
		return nil
	}
	if !setAuto {
		cfg.Log.WarnContext(ctx, "your presence is manually set to away, Slack will show you as away; use -set-auto to change it to automatic")
		return nil
	}
	if err := api.SetUserPresenceContext(ctx, "auto"); err != nil {
		return fmt.Errorf("set presence to auto: %w", err)
	}
	cfg.Log.InfoContext(ctx, "presence set to automatic")
	return nil
}

// wsHeader returns the websocket handshake headers that mimic the Slack web
// client.
func wsHeader(cookies []*http.Cookie) http.Header {
	h := http.Header{}
	h.Set("Origin", "https://app.slack.com")
	h.Set("User-Agent", slackauth.DefaultUserAgent)
	if len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, (&http.Cookie{Name: c.Name, Value: c.Value}).String())
		}
		h.Set("Cookie", strings.Join(parts, "; "))
	}
	return h
}
