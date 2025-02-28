package control

import (
	"context"
	"log/slog"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/processor"
)

// Option is a functional option for the Controller.
type Option func(*options)

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
func (o *options) newUserCollector(ctx context.Context) *userCollector {
	return &userCollector{
		ctx:   ctx,
		ts:    o.tf,
		users: make([]slack.User, 0, 100),
	}
}
