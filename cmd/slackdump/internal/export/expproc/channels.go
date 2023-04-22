package expproc

import (
	"context"

	"github.com/rusq/slackdump/v2/logger"
	"github.com/slack-go/slack"
)

type Channels struct {
	*baseproc
	fn func(c []slack.Channel) error
}

func NewChannels(dir string, fn func(c []slack.Channel) error) (*Channels, error) {
	p, err := newBaseProc(dir, "channels")
	if err != nil {
		return nil, err
	}
	return &Channels{baseproc: p, fn: fn}, nil
}

// Channels is called for each channel chunk that is retrieved.  Then, the
// function calls the function passed in to the constructor for the channel
// slice.
func (cp *Channels) Channels(ctx context.Context, channels []slack.Channel) error {
	lg := logger.FromContext(ctx)
	if lg.IsDebug() {
		names := make([]string, len(channels))
		for i, ch := range channels {
			names[i] = ch.Name
		}
		lg.Debug("channels names", names)
	}

	if err := cp.fn(channels); err != nil {
		return err
	}
	if err := cp.baseproc.Channels(ctx, channels); err != nil {
		return err
	}
	return nil
}
