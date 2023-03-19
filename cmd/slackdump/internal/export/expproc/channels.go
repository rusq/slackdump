package expproc

import (
	"context"

	"github.com/slack-go/slack"
)

type Channels struct {
	*baseproc
	fn func(c []slack.Channel) error
}

func NewChannels(dir string, fn func(c []slack.Channel) error) (*Channels, error) {
	p, err := newBaseProc(dir, "channels.json")
	if err != nil {
		return nil, err
	}
	return &Channels{baseproc: p}, nil
}

// Channels is called for each channel chunk that is retrieved.  Then, the
// function calls the function passed in to the constructor for the channel
// slice.
func (cp *Channels) Channels(ctx context.Context, channels []slack.Channel) error {
	if err := cp.baseproc.Channels(ctx, channels); err != nil {
		return err
	}
	return cp.fn(channels)
}
