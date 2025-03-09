package dirproc

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// Channels is a processor that writes the channel information into the
// channels file.
type Channels struct {
	*dirproc
}

// NewChannels creates a new Channels processor.  fn is called for each
// channel chunk that is retrieved.  The function is called before the chunk
// is processed by the recorder.
func NewChannels(dir *chunk.Directory) (*Channels, error) {
	p, err := newDirProc(dir, chunk.FChannels)
	if err != nil {
		return nil, err
	}
	return &Channels{dirproc: p}, nil
}

// Channels is called for each channel chunk that is retrieved.  Then, the
// function calls the function passed in to the constructor for the channel
// slice.
func (cp *Channels) Channels(ctx context.Context, channels []slack.Channel) error {
	if err := cp.dirproc.Channels(ctx, channels); err != nil {
		return err
	}
	return nil
}
