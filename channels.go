package slackdump

// In this file: channel/conversations and thread related code.

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rusq/dlog"
	"github.com/slack-go/slack"
)

// Channels keeps slice of channels
type Channels []slack.Channel

// GetChannels list all conversations for a user.  `chanTypes` specifies the
// type of messages to fetch.  See github.com/rusq/slack docs for possible
// values.  If large number of channels is to be returned, consider using
// StreamChannels.
func (sd *SlackDumper) GetChannels(ctx context.Context, chanTypes ...string) (Channels, error) {
	var allChannels Channels
	if err := sd.getChannels(ctx, chanTypes, func(cc []slack.Channel) error {
		allChannels = append(allChannels, cc...)
		return nil
	}); err != nil {
		return allChannels, err
	}
	return allChannels, nil
}

// StreamChannels requests the channels from the API and sends them over the channelC channel.
func (sd *SlackDumper) StreamChannels(ctx context.Context, chanTypes []string, cb func(ch slack.Channel) error) error {
	return sd.getChannels(ctx, chanTypes, func(chans []slack.Channel) error {
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
func (sd *SlackDumper) getChannels(ctx context.Context, chanTypes []string, cb func([]slack.Channel) error) error {
	ctx, task := trace.NewTask(ctx, "getChannels")
	defer task.End()

	limiter := newLimiter(tier2, sd.options.Tier2Burst, int(sd.options.Tier2Boost))

	if chanTypes == nil {
		chanTypes = AllChanTypes
	}

	params := &slack.GetConversationsParameters{Types: chanTypes, Limit: sd.options.ChannelsPerReq}
	fetchStart := time.Now()
	var total int
	for i := 1; ; i++ {
		var (
			chans   []slack.Channel
			nextcur string
		)
		reqStart := time.Now()
		if err := withRetry(ctx, limiter, sd.options.Tier3Retries, func() error {
			var err error
			trace.WithRegion(ctx, "GetConversationsContext", func() {
				chans, nextcur, err = sd.client.GetConversationsContext(ctx, params)
			})
			return err

		}); err != nil {
			return err
		}

		if err := cb(chans); err != nil {
			return err
		}
		total += len(chans)

		dlog.Printf("channels request #%5d, fetched: %4d, total: %8d (speed: %6.2f/sec, avg: %6.2f/sec)\n",
			i, len(chans), total,
			float64(len(chans))/float64(time.Since(reqStart).Seconds()),
			float64(total)/float64(time.Since(fetchStart).Seconds()),
		)

		if nextcur == "" {
			dlog.Printf("channels fetch complete, total: %d channels", total)
			break
		}

		params.Cursor = nextcur

		if err := limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ToText outputs Channels cs to io.Writer w in Text format.
func (cs Channels) ToText(w io.Writer, sd *SlackDumper) (err error) {
	const strFormat = "%s\t%s\t%s\t%s\n"
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer writer.Flush()
	fmt.Fprintf(writer, strFormat, "ID", "Arch", "Saved", "What")
	for i, ch := range cs {
		who := sd.whoThisChannelFor(&ch)
		archived := "-"
		if cs[i].IsArchived || sd.IsUserDeleted(ch.User) {
			archived = "arch"
		}
		saved := "-"
		if _, err := os.Stat(ch.ID + ".json"); err == nil {
			saved = "saved"
		}

		fmt.Fprintf(writer, strFormat, ch.ID, archived, saved, who)
	}
	return nil
}

// whoThisChannelFor return the proper name of the addressee of the channel
// Parameters: channel and userIdMap - mapping slackID to users
func (sd *SlackDumper) whoThisChannelFor(channel *slack.Channel) (who string) {
	switch {
	case channel.IsIM:
		who = "@" + sd.username(channel.User)
	case channel.IsMpIM:
		who = strings.Replace(channel.Purpose.Value, " messaging with", "", -1)
	case channel.IsPrivate:
		who = "🔒 " + channel.NameNormalized
	default:
		who = "#" + channel.NameNormalized
	}
	return who
}

// username tries to resolve the username by ID. If the internal users map is not
// initialised, it will return the ID, otherwise, if the user is not found in
// cache, it will assume that the user is external, and return the ID with
// "external" prefix.
func (sd *SlackDumper) username(id string) string {
	if sd.UserIndex == nil {
		// no user cache, use the IDs.
		return id
	}
	user, ok := sd.UserIndex[id]
	if !ok {
		return "<external>:" + id
	}
	return user.Name
}
