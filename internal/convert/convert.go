// Package convert implements conversions to different Slackdump formats.
package convert

import (
	"fmt"
	"log/slog"

	"github.com/rusq/slack"
)

type Option func(*options)

type options struct {
	// includeFiles is a flag to include files in the export
	includeFiles bool
	// includeAvatars is a flag to include avatars in the export
	includeAvatars bool
	// srcFileLoc should return the file location within the source directory.
	srcFileLoc func(*slack.Channel, *slack.File) string
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

// WithIncludeAvatars sets the IncludeAvataars option.
func WithIncludeAvatars(b bool) Option {
	return func(c *options) {
		c.includeAvatars = b
	}
}

// WithSrcFileLoc sets the SrcFileLoc function.
func WithSrcFileLoc(fn func(*slack.Channel, *slack.File) string) Option {
	return func(c *options) {
		if fn != nil {
			c.srcFileLoc = fn
		}
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
		if o.srcFileLoc == nil {
			return fmt.Errorf(format, "source", ErrNoLocFunction)
		}
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
