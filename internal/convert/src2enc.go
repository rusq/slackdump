// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package convert

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"slices"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
	"github.com/rusq/slackdump/v4/source"
)

// SourceEncoder allows to convert any source to a chunked format.
type SourceEncoder struct {
	src  source.Sourcer
	enc  chunk.Encoder
	fsa  fsadapter.FS // FS for files and avatars.
	opts options
}

func NewSourceEncoder(src source.Sourcer, fsa fsadapter.FS, enc chunk.Encoder, opts ...Option) *SourceEncoder {
	e := &SourceEncoder{
		src: src,
		enc: enc,
		fsa: fsa,
		opts: options{
			trgFileLoc: source.MattermostFilepath,
			lg:         slog.Default(),
		},
	}
	for _, o := range opts {
		o(&e.opts)
	}
	return e
}

func (s *SourceEncoder) Convert(ctx context.Context) error {
	rec := chunk.NewCustomRecorder(s.enc)
	if err := encodeWorkspaceInfo(ctx, rec, s.src); err != nil {
		return fmt.Errorf("workspace info: %w", err)
	}
	if err := encodeChannels(ctx, rec, s.src); err != nil {
		return fmt.Errorf("channels: %w", err)
	}

	var us processor.Users = rec
	if s.opts.includeAvatars && s.src.Avatars().Type() != source.STnone {
		acw := avatarcopywrapper{
			fsa:  s.fsa,
			avst: s.src.Avatars(),
		}
		// add a wrapper to the user processor to extract and copy avatar
		// images
		us = processor.JoinUsers(rec, &acw)
	}
	if err := encodeUsers(ctx, us, s.src); err != nil {
		return fmt.Errorf("users: %w", err)
	}

	var cp processor.Conversations = rec
	if s.opts.includeFiles && s.src.Files().Type() != source.STnone {
		fc := NewFileCopier(s.src, s.fsa, source.MattermostFilepath, s.opts.includeFiles)
		cp = processor.PrependFiler(rec, &filecopywrapper{
			fc:           fc,
			ignoreErrors: s.opts.ignoreCopyErrors,
		})
	}
	channels, err := s.src.Channels(ctx)
	if err != nil {
		return err
	}
	if err := encodeAllChannelMsg(ctx, cp, s.src, channels); err != nil {
		return fmt.Errorf("messages: %w", err)
	}
	if cm, ok := processor.AsCanvasMessenger(cp); ok {
		if err := encodeAllCanvasMessages(ctx, cp, cm, s.src, channels); err != nil {
			return fmt.Errorf("canvas messages: %w", err)
		}
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
		if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
			return nil
		}
		return err
	}
	return rec.WorkspaceInfo(ctx, wi)
}

func encodeAllChannelMsg(ctx context.Context, rec processor.Conversations, src source.Sourcer, channels []slack.Channel) error {
	for _, ch := range channels {
		if err := encodeMessages(ctx, rec, src, &ch); err != nil {
			if errors.Is(err, source.ErrNotFound) {
				slog.DebugContext(ctx, "encodeMessages", "channel", ch.ID, "error", err)
				continue
			}
			return err
		}
		// write channel information only for channels that have messages
		if err := rec.ChannelInfo(ctx, &ch, ""); err != nil {
			return err
		}
	}
	return nil
}

func encodeMessages(ctx context.Context, rec processor.Conversations, src source.Sourcer, ch *slack.Channel) error {
	messages, err := src.AllMessages(ctx, ch.ID)
	if err != nil {
		return err
	}

	var (
		chunk   = make([]slack.Message, 0, defaultChunkSize)
		threads = 0
	)
	for m, err := range messages {
		if err != nil {
			return fmt.Errorf("iterator for %s: %w", ch.ID, err)
		}
		chunk = append(chunk, m)
		if structures.IsThreadStart(&m) {
			if err := encodeThreadMessages(ctx, rec, src, ch, &m, m.Timestamp); err != nil {
				if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
					slog.DebugContext(ctx, "found thread, but no data for it", "channel", ch.ID, "thread", m.Timestamp, "reason", err)
				}
			} else {
				// only increase if the thread was found
				threads++
			}
		}
		if len(chunk) == defaultChunkSize {
			if err := rec.Messages(ctx, ch.ID, threads, false, chunk); err != nil {
				return err
			}
			chunk = make([]slack.Message, 0, defaultChunkSize)
			threads = 0
		}
		if len(m.Files) > 0 {
			if err := rec.Files(ctx, ch, m, m.Files); err != nil {
				return err
			}
		}
	}
	// flush
	if err := rec.Messages(ctx, ch.ID, threads, true, chunk); err != nil {
		return err
	}

	return nil
}

func encodeThreadMessages(ctx context.Context, rec processor.Conversations, src source.Sourcer, ch *slack.Channel, par *slack.Message, threadTS string) error {
	messages, err := src.AllThreadMessages(ctx, ch.ID, threadTS)
	if err != nil {
		return err
	}

	chunk := make([]slack.Message, 0, defaultChunkSize)
	for m, err := range messages {
		if err != nil {
			return fmt.Errorf("iterator for %s:%s: %w", ch.ID, threadTS, err)
		}
		chunk = append(chunk, m)
		if len(chunk) == defaultChunkSize {
			if err := rec.ThreadMessages(ctx, ch.ID, *par, false, false, chunk); err != nil {
				return err
			}
			chunk = make([]slack.Message, 0, defaultChunkSize)
		}
		if len(m.Files) > 0 {
			if err := rec.Files(ctx, ch, m, m.Files); err != nil {
				return err
			}
		}
	}
	// flush
	if err := rec.ThreadMessages(ctx, ch.ID, *par, false, true, chunk); err != nil {
		return err
	}

	return nil
}

type canvasSource interface {
	CanvasMessages(ctx context.Context, hiddenChannelID string) (iter.Seq2[slack.Message, error], error)
	CanvasThreadMessages(ctx context.Context, hiddenChannelID, threadTS string) (iter.Seq2[slack.Message, error], error)
}

func encodeAllCanvasMessages(ctx context.Context, rec processor.Conversations, cm processor.CanvasMessenger, src source.Sourcer, channels []slack.Channel) error {
	cs, ok := src.(canvasSource)
	if !ok {
		return nil
	}
	for i := range channels {
		fileID, ok := structures.CanvasFileID(&channels[i])
		if !ok {
			continue
		}
		hiddenChannelID, ok := structures.CanvasChannelID(fileID)
		if !ok {
			continue
		}
		if err := encodeCanvasMessages(ctx, rec, cm, cs, &channels[i], hiddenChannelID); err != nil {
			return err
		}
	}
	return nil
}

func encodeCanvasMessages(ctx context.Context, rec processor.Conversations, cm processor.CanvasMessenger, src canvasSource, owner *slack.Channel, hiddenChannelID string) error {
	it, err := src.CanvasMessages(ctx, hiddenChannelID)
	if err != nil {
		return err
	}
	roots := make([]slack.Message, 0, defaultChunkSize)
	numThreads := 0
	flush := func(isLast bool) error {
		if err := cm.CanvasMessages(ctx, hiddenChannelID, numThreads, isLast, roots); err != nil {
			return err
		}
		roots = make([]slack.Message, 0, defaultChunkSize)
		numThreads = 0
		return nil
	}
	for root, err := range it {
		if err != nil {
			return fmt.Errorf("iterator for canvas %s: %w", hiddenChannelID, err)
		}
		if root.ThreadTimestamp == "" {
			root.ThreadTimestamp = root.Timestamp
		}
		roots = append(roots, root)
		if root.ReplyCount > 0 {
			found, err := encodeCanvasThreadMessages(ctx, rec, cm, src, owner, hiddenChannelID, &root)
			if err != nil {
				return err
			}
			if found {
				numThreads++
			}
		}
		if len(roots) == defaultChunkSize {
			if err := flush(false); err != nil {
				return err
			}
		}
		if len(root.Files) > 0 {
			if err := rec.Files(ctx, owner, root, root.Files); err != nil {
				return err
			}
		}
	}
	return flush(true)
}

func encodeCanvasThreadMessages(ctx context.Context, rec processor.Conversations, cm processor.CanvasMessenger, src canvasSource, owner *slack.Channel, hiddenChannelID string, parent *slack.Message) (bool, error) {
	it, err := src.CanvasThreadMessages(ctx, hiddenChannelID, parent.ThreadTimestamp)
	if err != nil {
		return false, err
	}
	messages := make([]slack.Message, 0, defaultChunkSize)
	found := false
	for m, err := range it {
		if err != nil {
			return false, fmt.Errorf("iterator for canvas %s:%s: %w", hiddenChannelID, parent.ThreadTimestamp, err)
		}
		found = true
		messages = append(messages, m)
		if len(messages) == defaultChunkSize {
			if err := cm.CanvasThreadMessages(ctx, hiddenChannelID, *parent, false, messages); err != nil {
				return false, err
			}
			messages = make([]slack.Message, 0, defaultChunkSize)
		}
		if len(m.Files) > 0 {
			if err := rec.Files(ctx, owner, m, m.Files); err != nil {
				return false, err
			}
		}
	}
	if !found {
		return false, nil
	}
	if err := cm.CanvasThreadMessages(ctx, hiddenChannelID, *parent, true, messages); err != nil {
		return false, err
	}
	return true, nil
}
