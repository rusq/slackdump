package convert

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"slices"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

// Source encoder allows to convert any source to a chunked format.
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

type filecopywrapper struct {
	fc copier
}

func (f *filecopywrapper) Files(_ context.Context, ch *slack.Channel, parent slack.Message, _ []slack.File) error {
	return f.fc.Copy(ch, &parent)
}

func (f *filecopywrapper) Close() error {
	return nil
}

type avatarcopywrapper struct {
	fsa  fsadapter.FS
	avst source.Storage
}

func (a *avatarcopywrapper) Users(ctx context.Context, users []slack.User) error {
	if a.avst.Type() == source.STnone {
		return nil
	}
	for _, u := range users {
		if err := a.copyAvatar(u); err != nil {
			return err
		}
	}
	return nil
}

func (a *avatarcopywrapper) Close() error {
	return nil
}

func (a *avatarcopywrapper) copyAvatar(u slack.User) error {
	fsys := a.avst.FS()
	srcloc, err := a.avst.File(source.AvatarParams(&u))
	if err != nil {
		return err
	}
	dstloc := filepath.Join(chunk.AvatarsDir, filepath.Join(source.AvatarParams(&u)))
	src, err := fsys.Open(srcloc)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := a.fsa.Create(dstloc)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}

func (s *SourceEncoder) Convert(ctx context.Context) error {
	rec := chunk.NewCustomRecorder(s.enc)
	if err := encodeWorkspaceInfo(ctx, rec, s.src); err != nil {
		return err
	}
	if err := encodeChannels(ctx, rec, s.src); err != nil {
		return err
	}

	var us processor.Users = rec
	if s.opts.includeAvatars && s.src.Avatars().Type() != source.STnone {
		// TODO: implement
	}
	if err := encodeUsers(ctx, us, s.src); err != nil {
		return err
	}

	var cp processor.Conversations = rec
	if s.opts.includeFiles && s.src.Files().Type() != source.STnone {
		fc := NewFileCopier(s.src, s.fsa, source.MattermostFilepath, s.opts.includeFiles)
		cp = processor.PrependFiler(rec, &filecopywrapper{fc})
	}
	channels, err := s.src.Channels(ctx)
	if err != nil {
		return err
	}
	if err := encodeAllChannelMsg(ctx, cp, s.src, channels); err != nil {
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
		if errors.Is(err, source.ErrNotFound) || errors.Is(err, source.ErrNotSupported) {
			return nil
		}
		return err
	}
	return rec.WorkspaceInfo(ctx, wi)
}

func encodeAllChannelMsg(ctx context.Context, rec processor.Conversations, src source.Sourcer, channels []slack.Channel) error {
	for _, c := range channels {
		if err := encodeMessages(ctx, rec, src, &c); err != nil {
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
				return err
			}
			threads++
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
