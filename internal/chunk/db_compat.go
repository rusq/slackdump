package chunk

import (
	"context"

	"github.com/rusq/slack"
)

// db source compatibility layer

func (d *Directory) ChannelInfo(ctx context.Context, id string) (*slack.Channel, error) {
	f, err := d.Open(ToFileID(id, "", false))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ChannelInfo(id)
}
