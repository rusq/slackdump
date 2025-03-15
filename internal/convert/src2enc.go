package convert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

// Source encoder allows to convert any source to a chunked format.
type SourceEncoder struct {
	src source.Sourcer
	enc chunk.Encoder
	opt options
}

func NewSourceEncoder(src source.Sourcer, enc chunk.Encoder, opts ...Option) *SourceEncoder {
	e := &SourceEncoder{
		src: src,
		enc: enc,
		opt: options{
			srcFileLoc: src.Files().FilePath,
			trgFileLoc: source.MattermostFilepath,
			lg:         slog.Default(),
		},
	}
	return e
}

func (s *SourceEncoder) Convert(ctx context.Context) error {
	rec := chunk.NewCustomRecorder(s.enc)
	// TODO: files and avatars
	if err := encodeChannels(ctx, rec, s.src); err != nil {
		return err
	}
	if err := encodeUsers(ctx, rec, s.src); err != nil {
		return err
	}
	if err := encodeWorkspaceInfo(ctx, rec, s.src); err != nil {
		return err
	}

	channels, err := s.src.Channels(ctx)
	if err != nil {
		return err
	}
	if err := encodeAllChannelMsg(ctx, rec, s.src, channels); err != nil {
		return err
	}
	return nil
}

const (
	defaultChunkSize = 100
)

func encodeChannels(ctx context.Context, rec processor.Channels, src source.Sourcer) error {
	channels, err := src.Channels(ctx)
	if err != nil {
		return err
	}
	for ch := range slices.Chunk(channels, defaultChunkSize) {
		if err := rec.Channels(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func encodeUsers(ctx context.Context, rec processor.Users, src source.Sourcer) error {
	users, err := src.Users(ctx)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return nil
		}
		return err
	}
	for u := range slices.Chunk(users, defaultChunkSize) {
		if err := rec.Users(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

func encodeWorkspaceInfo(ctx context.Context, rec processor.WorkspaceInfo, src source.Sourcer) error {
	wi, err := src.WorkspaceInfo(ctx)
	if err != nil {
		if err == source.ErrNotFound {
			return nil
		}
		return err
	}
	return rec.WorkspaceInfo(ctx, wi)
}

func encodeAllChannelMsg(ctx context.Context, rec processor.Conversations, src source.Sourcer, channels []slack.Channel) error {
	for _, c := range channels {
		if err := encodeMessages(ctx, rec, src, c.ID); err != nil {
			return err
		}
	}
	return nil
}

func encodeMessages(ctx context.Context, rec processor.Conversations, src source.Sourcer, channelID string) error {
	messages, err := src.AllMessages(ctx, channelID)
	if err != nil {
		return err
	}

	var (
		chunk   = make([]slack.Message, 0, defaultChunkSize)
		threads = 0
	)
	for m, err := range messages {
		if err != nil {
			return fmt.Errorf("iterator for %s: %w", channelID, err)
		}
		chunk = append(chunk, m)
		if structures.IsThreadStart(&m) {
			if err := encodeThreadMessages(ctx, rec, src, channelID, &m, m.Timestamp); err != nil {
				return err
			}
			threads++
		}
		if len(chunk) == defaultChunkSize {
			if err := rec.Messages(ctx, channelID, threads, false, chunk); err != nil {
				return err
			}
			chunk = make([]slack.Message, 0, defaultChunkSize)
			threads = 0
		}
	}
	// flush
	if err := rec.Messages(ctx, channelID, threads, true, chunk); err != nil {
		return err
	}

	return nil
}

func encodeThreadMessages(ctx context.Context, rec processor.Conversations, src source.Sourcer, channelID string, par *slack.Message, threadTS string) error {
	messages, err := src.AllThreadMessages(ctx, channelID, threadTS)
	if err != nil {
		return err
	}

	chunk := make([]slack.Message, 0, defaultChunkSize)
	for m, err := range messages {
		if err != nil {
			return fmt.Errorf("iterator for %s:%s: %w", channelID, threadTS, err)
		}
		chunk = append(chunk, m)
		if len(chunk) == defaultChunkSize {
			if err := rec.ThreadMessages(ctx, channelID, *par, false, false, chunk); err != nil {
				return err
			}
			chunk = make([]slack.Message, 0, defaultChunkSize)
		}
	}
	// flush
	if err := rec.ThreadMessages(ctx, channelID, *par, false, true, chunk); err != nil {
		return err
	}

	return nil
}
