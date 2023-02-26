package processor

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/chunk"
)

// Replay replays the chunks in the reader to the Conversationer in the order
// they were recorded.
func Replay(ctx context.Context, ep chunk.Player, prc Conversationer) error {
	return ep.ForEach(func(ev *chunk.Chunk) error {
		if ev == nil {
			return nil
		}
		return emit(ctx, prc, *ev)
	})
}

// emit emits the chunk to the channeler.
func emit(ctx context.Context, prc Conversationer, evt chunk.Chunk) error {
	switch evt.Type {
	case chunk.CMessages:
		if err := prc.Messages(ctx, evt.ChannelID, evt.Messages); err != nil {
			return err
		}
	case chunk.CThreadMessages:
		if err := prc.ThreadMessages(ctx, evt.ChannelID, *evt.Parent, evt.Messages); err != nil {
			return err
		}
	case chunk.CFiles:
		if err := prc.Files(ctx, evt.ChannelID, *evt.Parent, evt.IsThreadMessage, evt.Files); err != nil {
			return err
		}
	}
	return nil
}
