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
	"log/slog"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/convert/transform"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/source"
)

type DumpConverter struct {
	src       source.Sourcer
	fsa       fsadapter.FS
	lg        *slog.Logger
	withFiles bool
}

type DumpOption func(*DumpConverter)

func DumpWithIncludeFiles(b bool) DumpOption {
	return func(s *DumpConverter) {
		s.withFiles = b
	}
}

func DumpWithLogger(log *slog.Logger) DumpOption {
	return func(s *DumpConverter) {
		s.lg = log
	}
}

// NewToDump creates a new dump converter.
func NewToDump(src source.Sourcer, trg fsadapter.FS, opts ...DumpOption) *DumpConverter {
	std := &DumpConverter{
		src: src,
		fsa: trg,
	}
	for _, opt := range opts {
		opt(std)
	}
	return std
}

func (d *DumpConverter) Convert(ctx context.Context) error {
	tfopts := []transform.DumpOption{
		transform.DumpWithLogger(d.lg),
	}
	if d.withFiles && d.src.Files().Type() != source.STnone {
		fh := &fileHandler{
			fc: NewFileCopier(d.src, d.fsa, source.DumpFilepath, d.withFiles),
		}
		tfopts = append(tfopts, transform.DumpWithPipeline(fh.copyFiles))
	}
	conv, err := transform.NewDump(
		d.fsa,
		d.src,
		tfopts...,
	)
	if err != nil {
		return err
	}
	return convert(ctx, d.src, conv)
}

type fileHandler struct {
	fc copier
}

//go:generate mockgen -destination=mock_convert/mock_copier.go . copier
type copier interface {
	Copy(*slack.Channel, *slack.Message) error
}

// copyFiles is a pipeline function that extracts files from messages and
// calls the file copier.
func (f *fileHandler) copyFiles(channelID string, _ string, mm []slack.Message) error {
	for _, m := range mm {
		if err := f.fc.Copy(structures.ChannelFromID(channelID), &m); err != nil {
			return err
		}
	}
	return nil
}
