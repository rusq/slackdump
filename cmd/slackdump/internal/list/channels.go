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

package list

import (
	"context"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/cache"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/types"
)

var CmdListChannels = &base.Command{
	Run:        runListChannels,
	UsageLine:  "slackdump list channels [flags] [filename]",
	PrintFlags: true,
	FlagMask:   flagMask &^ cfg.OmitChannelTypesFlag &^ cfg.OmitMemberOnlyFlag,
	Short:      "list workspace channels",
	Long: fmt.Sprintf(`
# List Channels Command

Lists all visible channels for the currently logged in user.  The list
includes all public and private channels, groups, and private messages (DMs),
including archived ones.

Please note that it may take a while to retrieve all channels, if your
workspace has lots of them.

The channels are cached, and the cache is valid for %s.  Use the -no-chan-cache
and -chan-cache-retention flags to control the cache behavior.
`+sectListFormat, chanFlags.cache.Retention),

	RequireAuth: true,
}

type (
	channelOptions struct {
		resolveUsers bool
		cache        cacheOpts
	}

	cacheOpts struct {
		Enabled   bool
		Retention time.Duration
		Filename  string
	}
)

var chanFlags = channelOptions{
	resolveUsers: false,
	cache: cacheOpts{
		Enabled:   false,
		Retention: 20 * time.Minute,
		Filename:  "channels.json",
	},
}

func init() {
	CmdListChannels.Wizard = wizChannels

	CmdListChannels.Flag.BoolVar(&chanFlags.cache.Enabled, "no-chan-cache", chanFlags.cache.Enabled, "disable channel cache")
	CmdListChannels.Flag.DurationVar(&chanFlags.cache.Retention, "chan-cache-retention", chanFlags.cache.Retention, "channel cache retention time.  After this time, the cache is considered stale and will be refreshed.")
	CmdListChannels.Flag.BoolVar(&chanFlags.resolveUsers, "resolve", chanFlags.resolveUsers, "resolve user IDs to names")
}

func runListChannels(ctx context.Context, cmd *base.Command, args []string) error {
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	l := &channels{
		opts:   chanFlags,
		common: commonFlags,
	}

	return list(ctx, sess, l, filename)
}

type channels struct {
	channels types.Channels
	users    types.Users

	opts   channelOptions
	common commonOpts
}

func (l *channels) Type() string {
	return "channels"
}

func (l *channels) Data() types.Channels {
	return l.channels
}

func (l *channels) Users() []slack.User {
	return l.users
}

func (l *channels) Retrieve(ctx context.Context, sess *slackdump.Session, m *cache.Manager) error {
	ctx, task := trace.NewTask(ctx, "channels.List")
	defer task.End()
	lg := cfg.Log

	teamID := sess.Info().TeamID

	usersc := make(chan []slack.User)
	go func() {
		defer close(usersc)
		if l.opts.resolveUsers {
			lg.InfoContext(ctx, "getting users to resolve DM names")
			u, err := fetchUsers(ctx, sess, m, cfg.NoUserCache, teamID)
			if err != nil {
				lg.ErrorContext(ctx, "error getting users to resolve DM names", "error", err)
				return
			}
			usersc <- u
		}
	}()

	if l.opts.cache.Enabled {
		var err error
		l.channels, err = m.LoadChannels(teamID, l.opts.cache.Retention)
		if err == nil {
			l.users = <-usersc
			return nil
		}
	}
	p := slackdump.GetChannelsParameters{
		ChannelTypes:   cfg.ChannelTypes,
		OnlyMyChannels: cfg.MemberOnly,
	}

	var cc []slack.Channel
	for chans, err := range sess.StreamChannelsEx(ctx, p) {
		if err != nil {
			return fmt.Errorf("error getting channels: %w", err)
		}
		if cfg.MemberOnly {
			for _, ch := range chans {
				if structures.IsMember(&ch) {
					cc = append(cc, ch)
				}
			}
		} else {
			cc = append(cc, chans...)
		}
	}
	l.channels = cc
	l.users = <-usersc
	if err := m.CacheChannels(teamID, cc); err != nil {
		lg.WarnContext(ctx, "failed to cache channels (ignored)", "error", err)
	}
	return nil
}

func (l *channels) Len() int {
	return len(l.channels)
}
