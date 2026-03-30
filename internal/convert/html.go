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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"path"
	"strings"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/internal/viewer"
	"github.com/rusq/slackdump/v4/internal/viewer/renderer"
	"github.com/rusq/slackdump/v4/source"
)

type HTMLConverter struct {
	src source.Sourcer
	trg fsadapter.FS
	lg  *slog.Logger
}

func NewToHTML(src source.Sourcer, trg fsadapter.FS, opts ...Option) *HTMLConverter {
	c := &HTMLConverter{
		src: src,
		trg: trg,
		lg:  slog.Default(),
	}
	cfg := options{lg: c.lg}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.lg != nil {
		c.lg = cfg.lg
	}
	return c
}

func (c *HTMLConverter) Validate() error {
	if c.src == nil || c.trg == nil {
		return errors.New("convert: source and target must be set")
	}
	return nil
}

func (c *HTMLConverter) Convert(ctx context.Context) error {
	if err := c.Validate(); err != nil {
		return err
	}

	v, err := viewer.New(ctx, "", c.src, viewer.WithMode(renderer.ModeStatic))
	if err != nil {
		return err
	}

	if err := c.renderPage(ctx, v.RenderIndex, "index.html"); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	channels, err := c.src.Channels(ctx)
	if err != nil {
		return err
	}
	for _, ch := range channels {
		if err := c.renderPage(ctx, func(ctx context.Context, w io.Writer) error {
			return v.RenderChannel(ctx, ch.ID, w)
		}, channelPagePath(ch.ID)); err != nil {
			return fmt.Errorf("channel %s: %w", ch.ID, err)
		}

		threadRoots, err := c.threadRoots(ctx, ch.ID)
		if err != nil {
			return fmt.Errorf("channel %s threads: %w", ch.ID, err)
		}
		for _, threadTS := range threadRoots {
			if err := c.renderPage(ctx, func(ctx context.Context, w io.Writer) error {
				return v.RenderThread(ctx, ch.ID, threadTS, w)
			}, threadPagePath(ch.ID, threadTS)); err != nil {
				return fmt.Errorf("channel %s thread %s: %w", ch.ID, threadTS, err)
			}
		}

		if ch.Properties != nil && ch.Properties.Canvas.FileId != "" {
			if err := c.renderPage(ctx, func(ctx context.Context, w io.Writer) error {
				return v.RenderCanvas(ctx, ch.ID, w)
			}, canvasPagePath(ch.ID)); err != nil {
				return fmt.Errorf("channel %s canvas: %w", ch.ID, err)
			}
			if err := c.renderRaw(ctx, func(ctx context.Context, w io.Writer) error {
				return v.RenderCanvasContent(ctx, ch.ID, w)
			}, canvasContentPath(ch.ID)); err != nil && !errors.Is(err, source.ErrNotFound) && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("channel %s canvas content: %w", ch.ID, err)
			}
		}

		if err := c.copyChannelFiles(ctx, ch, threadRoots); err != nil {
			return fmt.Errorf("channel %s files: %w", ch.ID, err)
		}
	}

	users, err := c.src.Users(ctx)
	if err != nil {
		if !errors.Is(err, source.ErrNotFound) {
			return err
		}
		users = nil
	}
	for _, u := range users {
		if err := c.renderPage(ctx, func(ctx context.Context, w io.Writer) error {
			return v.RenderUser(ctx, u.ID, w)
		}, userPagePath(u.ID)); err != nil {
			return fmt.Errorf("user %s: %w", u.ID, err)
		}
	}
	if err := c.copyAvatars(users); err != nil {
		return fmt.Errorf("avatars: %w", err)
	}
	if err := c.copyStaticAssets(); err != nil {
		return fmt.Errorf("static assets: %w", err)
	}

	return nil
}

func (c *HTMLConverter) copyChannelFiles(ctx context.Context, ch slack.Channel, threadRoots []string) error {
	if c.src.Files().Type() == source.STnone {
		return nil
	}
	fc := NewFileCopier(c.src, c.trg, htmlFilePath, true)
	if err := c.copyFileSeq(ctx, fc, &ch, ch.ID, mustAllMessages(ctx, c.src, ch.ID)); err != nil {
		return err
	}
	for _, threadTS := range threadRoots {
		it, err := c.src.AllThreadMessages(ctx, ch.ID, threadTS)
		if err != nil {
			if errors.Is(err, source.ErrNotFound) {
				continue
			}
			return err
		}
		if err := c.copyFileSeq(ctx, fc, &ch, ch.ID, it); err != nil {
			return err
		}
	}
	return nil
}

func mustAllMessages(ctx context.Context, src source.Sourcer, channelID string) iter.Seq2[slack.Message, error] {
	it, err := src.AllMessages(ctx, channelID)
	if err != nil {
		return func(yield func(slack.Message, error) bool) {
			yield(slack.Message{}, err)
		}
	}
	return it
}

func (c *HTMLConverter) copyFileSeq(ctx context.Context, fc *FileCopier, ch *slack.Channel, channelID string, it iter.Seq2[slack.Message, error]) error {
	for msg, err := range it {
		if err != nil {
			if errors.Is(err, source.ErrNotFound) {
				return nil
			}
			return err
		}
		err = fc.Copy(ch, &msg)
		if err == nil {
			continue
		}
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, source.ErrNotFound) {
			c.lg.WarnContext(ctx, "skipping missing file asset", "channel", channelID, "ts", msg.Timestamp, "error", err)
			continue
		}
		return err
	}
	return nil
}

func (c *HTMLConverter) copyAvatars(users []slack.User) error {
	if c.src.Avatars().Type() == source.STnone {
		return nil
	}
	for _, u := range users {
		if u.Profile.ImageOriginal == "" {
			continue
		}
		userID, filename := source.AvatarParams(&u)
		srcPath, err := c.src.Avatars().File(userID, filename)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, source.ErrNotFound) {
				c.lg.Warn("skipping missing avatar asset", "user", u.ID, "error", err)
				continue
			}
			return err
		}
		if err := copy2trg(c.trg, htmlAvatarPath(userID, filename), c.src.Avatars().FS(), srcPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, source.ErrNotFound) {
				c.lg.Warn("skipping missing avatar asset", "user", u.ID, "error", err)
				continue
			}
			return err
		}
	}
	return nil
}

func (c *HTMLConverter) copyStaticAssets() error {
	staticFS := viewer.StaticFS()
	return fs.WalkDir(staticFS, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return copy2trg(c.trg, htmlStaticAssetPath(name), staticFS, name)
	})
}

func (c *HTMLConverter) renderPage(ctx context.Context, render func(context.Context, io.Writer) error, outputPath string) error {
	var buf bytes.Buffer
	if err := render(ctx, &buf); err != nil {
		return err
	}
	body := relativizeRootLinks(buf.Bytes(), outputPath)
	return c.trg.WriteFile(outputPath, body, 0o644)
}

func (c *HTMLConverter) renderRaw(ctx context.Context, render func(context.Context, io.Writer) error, outputPath string) error {
	var buf bytes.Buffer
	if err := render(ctx, &buf); err != nil {
		return err
	}
	return c.trg.WriteFile(outputPath, buf.Bytes(), 0o644)
}

func (c *HTMLConverter) threadRoots(ctx context.Context, channelID string) ([]string, error) {
	it, err := c.src.AllMessages(ctx, channelID)
	if err != nil {
		if errors.Is(err, source.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var roots []string
	for msg, err := range it {
		if err != nil {
			return nil, err
		}
		if structures.IsThreadStart(&msg) {
			roots = append(roots, msg.ThreadTimestamp)
		}
	}
	return roots, nil
}

func relativizeRootLinks(body []byte, outputPath string) []byte {
	prefix := relativePrefix(outputPath)
	if prefix == "" {
		prefix = ""
	}
	replacer := strings.NewReplacer(
		`="/`, `="`+prefix,
		`='/`, `='`+prefix,
		`url(/`, `url(`+prefix,
	)
	return []byte(replacer.Replace(string(body)))
}

func relativePrefix(outputPath string) string {
	dir := path.Dir(path.Clean(outputPath))
	if dir == "." || dir == "" {
		return ""
	}
	return strings.Repeat("../", strings.Count(dir, "/")+1)
}

func channelPagePath(channelID string) string {
	return path.Join("archives", channelID, "index.html")
}

func threadPagePath(channelID, threadTS string) string {
	return path.Join("archives", channelID, "threads", threadTS+".html")
}

func canvasPagePath(channelID string) string {
	return path.Join("archives", channelID, "canvas", "index.html")
}

func canvasContentPath(channelID string) string {
	return path.Join("archives", channelID, "canvas", "content.html")
}

func userPagePath(userID string) string {
	return path.Join("team", userID, "index.html")
}

func htmlFilePath(_ *slack.Channel, f *slack.File) string {
	return path.Join("files", f.ID, source.SanitizeFilename(f.Name))
}

func htmlAvatarPath(userID, filename string) string {
	return path.Join("avatars", userID, filename)
}

func htmlStaticAssetPath(name string) string {
	return path.Join("static", name)
}
