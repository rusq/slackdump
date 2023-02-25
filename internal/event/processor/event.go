package processor

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/event"
)

// Replay replays the events in the reader to the channeler in the order they
// were recorded.  It will reset the state of the Player.
func Replay(ctx context.Context, ep event.Player, prc Processor) error {
	return ep.ForEach(func(ev *event.Event) error {
		if ev == nil {
			return nil
		}
		return emit(ctx, prc, *ev)
	})
}

// emit emits the event to the channeler.
func emit(ctx context.Context, prc Processor, evt event.Event) error {
	switch evt.Type {
	case event.EMessages:
		if err := prc.Messages(ctx, evt.ChannelID, evt.Messages); err != nil {
			return err
		}
	case event.EThreadMessages:
		if err := prc.ThreadMessages(ctx, evt.ChannelID, *evt.Parent, evt.Messages); err != nil {
			return err
		}
	case event.EFiles:
		if err := prc.Files(ctx, evt.ChannelID, *evt.Parent, evt.IsThreadMessage, evt.Files); err != nil {
			return err
		}
	}
	return nil
}
