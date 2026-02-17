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

package slackdump

// In this file: channel/conversations and thread related code.

import (
	"context"
	"errors"
	"iter"
	"log/slog"
	"runtime/trace"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/network"
	"github.com/rusq/slackdump/v4/stream"
	"github.com/rusq/slackdump/v4/types"
)

// GetChannelsParameters holds the parameters for [GetChannelsEx] and
// [StreamChannelsEx] functions.
type GetChannelsParameters struct {
	// ChannelTypes allows to specify the channel types to fetch.  If the slice
	// is empty, all channel types will be fetched.
	ChannelTypes []string
	// OnlyMyChannels restricts the channels only to the channels that the user
	// is a member of.
	OnlyMyChannels bool
}

// GetChannels list all conversations for a user.  `chanTypes` specifies the
// type of channels to fetch.  See github.com/rusq/slack docs for possible
// values.  If large number of channels is to be returned, consider using
// [StreamChannelsEx].  It is a wrapper for [GetChannelsEx].
//
// Deprecated; Use [GetChannelsEx].  This function Will be removed in v5.
func (s *Session) GetChannels(ctx context.Context, chanTypes ...string) (types.Channels, error) {
	p := GetChannelsParameters{
		ChannelTypes:   chanTypes,
		OnlyMyChannels: false,
	}
	return s.GetChannelsEx(ctx, p)
}

// GetChannelsEx list all conversations for a user. GetChannelParameters should
// contain the fetch criteria. If large number of channels is to be returned,
// consider using [StreamChannelsEx].
func (s *Session) GetChannelsEx(ctx context.Context, p GetChannelsParameters) (types.Channels, error) {
	var allChannels types.Channels
	if err := s.getChannels(ctx, p, func(ctx context.Context, cc types.Channels) error {
		allChannels = append(allChannels, cc...)
		return nil
	}); err != nil {
		return allChannels, err
	}
	return allChannels, nil
}

// StreamChannels requests the channels from the API and calls the callback
// function cb for each.  It is a wrapper for [StreamChannelsEx].
//
// Deprecated: Use [StreamChannelsEx]. This function Will be removed in v5.
func (s *Session) StreamChannels(ctx context.Context, chanTypes []string, cb func(ch slack.Channel) error) error {
	p := GetChannelsParameters{
		ChannelTypes:   chanTypes,
		OnlyMyChannels: false,
	}
	for chans, err := range s.StreamChannelsEx(ctx, p) {
		if err != nil {
			return err
		}
		for _, ch := range chans {
			if err := cb(ch); err != nil {
				return err
			}
		}
	}
	return nil
}

// StreamChannelsEx requests the channels from the API and returns an iterator
// of channel chunks.
func (s *Session) StreamChannelsEx(ctx context.Context, p GetChannelsParameters) iter.Seq2[[]slack.Channel, error] {
	return func(yield func(ch []slack.Channel, err error) bool) {
		err := s.getChannels(ctx, p, func(ctx context.Context, chans types.Channels) error {
			if !yield(chans, nil) {
				return ErrStop
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, ErrStop) {
				return
			}
			_ = yield(nil, err)
		}
	}
}

type chanProcFunc func(ctx context.Context, ch types.Channels) error

func (f chanProcFunc) Channels(ctx context.Context, ch []slack.Channel) error {
	return f(ctx, ch)
}

// ErrStop instructs early stop to streaming function, when returned from a
// callback function.
var ErrStop = errors.New("stop")

// getChannels list all channels for a user.  `chanTypes` specifies
// the type of messages to fetch.  See github.com/rusq/slack docs for possible
// values.  If the cb function returns [ErrStop], the iteration will stop.
func (s *Session) getChannels(ctx context.Context, gcp GetChannelsParameters, cb chanProcFunc) error {
	ctx, task := trace.NewTask(ctx, "getChannels")
	defer task.End()

	if len(gcp.ChannelTypes) == 0 {
		gcp.ChannelTypes = AllChanTypes
	}

	st := s.Stream()
	params := &slack.GetConversationsParameters{Types: gcp.ChannelTypes, Limit: s.cfg.limits.Request.Channels}
	if err := st.ListChannelsEx(ctx, cb, params, gcp.OnlyMyChannels); err != nil {
		if errors.Is(err, ErrStop) {
			// early stop indicated
			return nil
		}

		if !shouldFallbackToListChannels(err) {
			return err
		}
		slog.DebugContext(ctx, "falling back to simple List Channels", "err", err)
		if err := st.ListChannels(ctx, cb, params); err != nil {
			if errors.Is(err, ErrStop) {
				// early stop indicated
				return nil
			}
			return err
		}
	}
	return nil
}

func shouldFallbackToListChannels(err error) bool {
	if errors.Is(err, stream.ErrOpNotSupported) {
		return true
	}
	return false
}

// GetChannelMembers returns a list of all lmembers in a channel.
func (sd *Session) GetChannelMembers(ctx context.Context, channelID string) ([]string, error) {
	var ids []string
	var cursor string
	for {
		var uu []string
		var next string
		if err := network.WithRetry(ctx, sd.limiter(network.Tier4), sd.cfg.limits.Tier4.Retries, func(ctx context.Context) error {
			var err error
			uu, next, err = sd.client.GetUsersInConversationContext(ctx, &slack.GetUsersInConversationParameters{
				ChannelID: channelID,
				Cursor:    cursor,
			})
			return err
		}); err != nil {
			return nil, err
		}
		ids = append(ids, uu...)

		if next == "" {
			break
		}
		cursor = next
	}
	return ids, nil
}
