package processor

import (
	"context"

	"github.com/rusq/slack"
)

type NopFiler struct{}

func (n *NopFiler) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {
	return nil
}
func (n *NopFiler) Close() error { return nil }

type NopAvatars struct{}

func (n *NopAvatars) Users(ctx context.Context, users []slack.User) error { return nil }
func (n *NopAvatars) Close() error                                        { return nil }

type NopChannels struct{}

func (NopChannels) Channels(ctx context.Context, ch []slack.Channel) error {
	return nil
}
