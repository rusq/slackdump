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
package control

import (
	"context"
	"log/slog"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/processor"
)

// Option is a functional option for the Controller.
type Option func(*options)

type options struct {
	// tf is the transformer of the chunk data. It may not be necessary, if
	// caller is not interested in transforming the data.
	tf ExportTransformer
	// files subprocessor, if not configured with options, it's a noop, as
	// it's not necessary for all use cases.
	filer processor.Filer
	// avp is avatar downloader (subprocessor), if not configured with options,
	// it's a noop, as it's not necessary
	avp processor.Avatars
	// lg is the logger
	lg *slog.Logger
	// flags
	flags Flags
}

// WithFiler configures the controller with a file subprocessor.
func WithFiler(f processor.Filer) Option {
	return func(c *options) {
		c.filer = f
	}
}

// WithAvatarProcessor configures the controller with an avatar downloader.
func WithAvatarProcessor(avp processor.Avatars) Option {
	return func(c *options) {
		c.avp = avp
	}
}

// WithFlags configures the controller with flags.
func WithFlags(f Flags) Option {
	return func(c *options) {
		c.flags = f
	}
}

// WithCoordinator configures the controller with a transformer.
func WithCoordinator(tf ExportTransformer) Option {
	return func(c *options) {
		if tf != nil {
			c.tf = tf
		}
	}
}

// WithLogger configures the controller with a logger.
func WithLogger(lg *slog.Logger) Option {
	return func(c *options) {
		if lg != nil {
			c.lg = lg
		}
	}
}

// helpers

// newUserCollector creates a new user collector.
func (o *options) newUserCollector(ctx context.Context, allowNoUsers bool) *userCollector {
	return &userCollector{
		ctx:        ctx,
		ts:         o.tf,
		users:      make([]slack.User, 0, 100),
		allowEmpty: allowNoUsers,
	}
}
