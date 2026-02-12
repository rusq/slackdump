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

// Package convert implements conversions to different Slackdump formats.  It
// is a layer on top of the transformer.

package convert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v4/source"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/convert/transform"
)

// Target is the interface for writing the target format.
type Target interface {
	// Convert should convert the data for the single channel and save it to
	// the target format.
	transform.Converter
	// Users should convert and write users.
	Users(ctx context.Context, uu []slack.User) error
	// Channels should converts and write channels.
	Channels(ctx context.Context, uu []slack.Channel) error
	// WorkspaceInfo writes workspace info.
	WorkspaceInfo(ctx context.Context, wi *slack.AuthTestResponse) error
}

type Option func(*options)

type options struct {
	// includeFiles is a flag to include files in the export
	includeFiles bool
	// includeAvatars is a flag to include avatars in the export
	includeAvatars bool
	// ignoreCopyErrors is a flag to ignore copy errors
	ignoreCopyErrors bool
	// trgFileLoc should return the file location within the target directory
	trgFileLoc func(*slack.Channel, *slack.File) string
	// avtrFileLoc should return the avatar file location.
	avtrFileLoc func(*slack.User) string
	// lg is the logger
	lg *slog.Logger
}

// WithIncludeFiles sets the IncludeFiles option.
func WithIncludeFiles(b bool) Option {
	return func(c *options) {
		c.includeFiles = b
	}
}

// WithIncludeAvatars sets the IncludeAvatars option.
func WithIncludeAvatars(b bool) Option {
	return func(c *options) {
		c.includeAvatars = b
	}
}

func WithIgnoreCopyErrors(b bool) Option {
	return func(c *options) {
		c.ignoreCopyErrors = b
	}
}

// WithTrgFileLoc sets the TrgFileLoc function.
func WithTrgFileLoc(fn func(*slack.Channel, *slack.File) string) Option {
	return func(c *options) {
		if fn != nil {
			c.trgFileLoc = fn
		}
	}
}

// WithLogger sets the logger.
func WithLogger(lg *slog.Logger) Option {
	return func(c *options) {
		if lg != nil {
			c.lg = lg
		}
	}
}

func (o *options) Validate() error {
	const format = "convert: internal error: %s: %w"
	if o.includeFiles {
		if o.trgFileLoc == nil {
			return fmt.Errorf(format, "target", ErrNoLocFunction)
		}
	}
	if o.includeAvatars {
		if o.avtrFileLoc == nil {
			return fmt.Errorf(format, "avatar", ErrNoLocFunction)
		}
	}
	return nil
}

// convert is a simple single-threaded conversion function, that, given
// a source and a target, converts the source data to the target format.
func convert(ctx context.Context, src source.Sourcer, trg Target) error {
	channels, err := src.Channels(ctx)
	if err != nil {
		return err
	}
	if err := trg.Channels(ctx, channels); err != nil {
		return err
	}
	for _, c := range channels {
		// TODO: having FileID is an atavism, should be a channelID at least.
		//       check usages, if it's possible to change.
		if err := trg.Convert(ctx, c.ID, ""); err != nil {
			return err
		}
	}

	users, err := src.Users(ctx)
	if err == nil {
		if err := trg.Users(ctx, users); err != nil {
			return err
		}
	} else if !errors.Is(err, source.ErrNotFound) {
		return err
	}

	wi, err := src.WorkspaceInfo(ctx)
	if err == nil {
		if err := trg.WorkspaceInfo(ctx, wi); err != nil {
			return err
		}
	} else if !errors.Is(err, source.ErrNotFound) {
		return err
	}

	return nil
}
