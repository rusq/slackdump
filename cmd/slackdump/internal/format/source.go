package format

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/trace"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/format"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/types"
)

// formatSrc formats the source with the given formatter and writes the result
// into fsa. If el is specified, it is used to filter the channels that are
// formatted.
func formatSrc(ctx context.Context, fsa fsadapter.FS, src source.Sourcer, formatter format.Formatter, el *structures.EntityList) error {
	ctx, task := trace.NewTask(ctx, "format")
	defer task.End()
	lg := cfg.Log

	// Get the source type and name.
	srcType, srcName := src.Type(), src.Name()

	fileext := formatter.Extension()

	lg.InfoContext(ctx, "source", "type", srcType.String(), "name", srcName)
	users, err := src.Users(ctx)
	if err != nil {
		lg.WarnContext(ctx, "users will not be resolved, no users in the source")
	} else {
		if err := withFile(fsa, "users"+fileext, func(w io.Writer) error {
			return formatter.Users(ctx, w, users)
		}); err != nil {
			return err
		}
	}

	channels, err := src.Channels(ctx)
	if err != nil {
		lg.WarnContext(ctx, "channels will not be formatted, no channels in the source")
	} else {
		if err := withFile(fsa, "channels"+fileext, func(w io.Writer) error {
			return formatter.Channels(ctx, w, users, channels)
		}); err != nil {
			return err
		}
	}

	// format messages
	for _, ch := range channels {
		if el != nil {
			ch, ok := el.Get(ch.ID)
			if !ok || !ch.Include {
				continue
			}
		}
		conv, err := getChannel(ctx, src, ch)
		if err != nil {
			if errors.Is(err, source.ErrNotFound) {
				continue
			}
			lg.WarnContext(ctx, "failed to format channel", "channel", ch.ID, "error", err)
			continue
		}
		if len(conv.Messages) == 0 {
			continue
		}
		if err := withFile(fsa, ch.ID+fileext, func(w io.Writer) error {
			return formatter.Conversation(ctx, w, users, &conv)
		}); err != nil {
			return err
		}
	}

	return nil
}

func getChannel(ctx context.Context, src source.Sourcer, ch slack.Channel) (types.Conversation, error) {
	// check for entity list inclusion
	conv := types.Conversation{
		ID:   ch.ID,
		Name: ch.Name,
	}
	it, err := src.AllMessages(ctx, ch.ID)
	if err != nil {
		return conv, err
	}
	for slackMsg, err := range it {
		if err != nil {
			return conv, err
		}
		msg := types.Message{Message: slackMsg}
		if structures.IsThreadStart(&slackMsg) && !structures.IsEmptyThread(&slackMsg) {
			thread, err := getThread(ctx, src, ch.ID, slackMsg.ThreadTimestamp)
			if err != nil {
				if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
					// thread not found or not supported,ignore
				} else {
					return conv, err
				}
			} else {
				msg.ThreadReplies = thread
			}
		}
		conv.Messages = append(conv.Messages, msg)
	}

	return conv, nil
}

func getThread(ctx context.Context, src source.Sourcer, chanID string, ts string) ([]types.Message, error) {
	itt, err := src.AllThreadMessages(ctx, chanID, ts)
	if err != nil {
		return nil, err
	}
	var mm []types.Message
	for threadMsg, err := range itt {
		if err != nil {
			return nil, err
		}
		mm = append(mm, types.Message{Message: threadMsg})
	}
	return mm[1:], nil
}

// withFile opens a file for writing and calls the provided callback function.
func withFile(fsa fsadapter.FS, filename string, cb func(w io.Writer) error) (err error) {
	// Open the w for writing.
	w, err := fsa.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() {
		err = errors.Join(err, w.Close())
	}()

	// Call the function with the opened file.
	if err := cb(w); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}

	return nil
}
