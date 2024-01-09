package dirproc

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/slack-go/slack"
)

type Channels struct {
	*baseproc
	fn func(c []slack.Channel) error
}

// NewChannels creates a new Channels processor.  fn is called for each
// channel chunk that is retrieved.  The function is called before the chunk
// is processed by the recorder.
func NewChannels(dir *chunk.Directory, fn func(c []slack.Channel) error) (*Channels, error) {
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
	if err := cp.fn(channels); err != nil {
		return err
	}
	if err := cp.baseproc.Channels(ctx, channels); err != nil {
		return err
	}
	return nil
}
