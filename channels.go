package slackdump

// In this file: channel/conversations and thread related code.

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/slack-go/slack"

	"github.com/rusq/slackdump/v2/internal/network"
	"github.com/rusq/slackdump/v2/types"
)

// GetChannels list all conversations for a user.  `chanTypes` specifies the
// type of messages to fetch.  See github.com/rusq/slack docs for possible
// values.  If large number of channels is to be returned, consider using
// StreamChannels.
func (s *Session) GetChannels(ctx context.Context, chanTypes ...string) (types.Channels, error) {
	var allChannels types.Channels
	if err := s.getChannels(ctx, chanTypes, func(cc types.Channels) error {
		allChannels = append(allChannels, cc...)
		return nil
	}); err != nil {
		return allChannels, err
	}
	return allChannels, nil
}

// StreamChannels requests the channels from the API and calls the callback
// function cb for each.
func (s *Session) StreamChannels(ctx context.Context, chanTypes []string, cb func(ch slack.Channel) error) error {
	return s.getChannels(ctx, chanTypes, func(chans types.Channels) error {
		for _, ch := range chans {
			if err := cb(ch); err != nil {
				return err
			}
		}
		return nil
	})
}

// getChannels list all conversations for a user.  `chanTypes` specifies
// the type of messages to fetch.  See github.com/rusq/slack docs for possible
// values
func (s *Session) getChannels(ctx context.Context, chanTypes []string, cb func(types.Channels) error) error {
	ctx, task := trace.NewTask(ctx, "getChannels")
	defer task.End()

	limiter := network.NewLimiter(network.Tier2, s.cfg.Limits.Tier2.Burst, int(s.cfg.Limits.Tier2.Boost))

	if chanTypes == nil {
		chanTypes = AllChanTypes
	}

	params := &slack.GetConversationsParameters{Types: chanTypes, Limit: s.cfg.Limits.Request.Channels}
	fetchStart := time.Now()
	var total int
	for i := 1; ; i++ {
		var (
			chans   []slack.Channel
			nextcur string
		)
		reqStart := time.Now()
		if err := withRetry(ctx, limiter, s.cfg.Limits.Tier3.Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationsContext", func() {
				chans, nextcur, err = s.client.GetConversationsContext(ctx, params)
			})
			return err

		}); err != nil {
			return err
		}

		if err := cb(chans); err != nil {
			return err
		}
		total += len(chans)

		s.l().Printf("channels request #%5d, fetched: %4d, total: %8d (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i, len(chans), total,
			float64(len(chans))/float64(time.Since(reqStart).Seconds()),
			float64(total)/float64(time.Since(fetchStart).Seconds()),
		)

		if nextcur == "" {
			s.l().Printf("channels fetch complete, total: %d channels", total)
			break
		}

		params.Cursor = nextcur

		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}
