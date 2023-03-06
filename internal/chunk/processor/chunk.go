package processor

import (
	"context"

	"github.com/rusq/slackdump/v2/internal/chunk"
)

// Replay replays the chunks in the reader to the Conversation in the order
// they were recorded.
func Replay(ctx context.Context, ep *chunk.Player, prc Conversations) error {
	return ep.ForEach(func(ev *chunk.Chunk) error {
		if ev == nil {
			return nil
		}
		return emit(ctx, prc, *ev)
	})
}

// emit emits the chunk to the Conversationer.
func emit(ctx context.Context, prc Conversations, ch chunk.Chunk) error {
	switch ch.Type {
	case chunk.CChannelInfo:
		if err := prc.ChannelInfo(ctx, ch.Channel, ch.IsThread); err != nil {
			return err
		}
	case chunk.CMessages:
		if err := prc.Messages(ctx, ch.ChannelID, ch.Messages); err != nil {
			return err
		}
	case chunk.CThreadMessages:
		if err := prc.ThreadMessages(ctx, ch.ChannelID, *ch.Parent, ch.Messages); err != nil {
			return err
		}
	case chunk.CFiles:
		if err := prc.Files(ctx, ch.ChannelID, *ch.Parent, ch.IsThread, ch.Files); err != nil {
			return err
		}
	}
	return nil
}
