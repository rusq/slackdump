package control

import (
	"context"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/processor"
)

// stream.go contains the overrides for the Streamer.

// usersCollector replaces the Users method of the Streamer
// with a method that gets the information for user IDs received on
// the userIDC channel and calls the Users processor method.
type userCollectingStreamer struct {
	Streamer
	userIDC       <-chan []string
	includeLabels bool
}

// Users is the override for the Streamer.Users method.
func (u *userCollectingStreamer) Users(ctx context.Context, proc processor.Users, opt ...slack.GetUsersOption) error {
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case ids, ok := <-u.userIDC:
			if !ok {
				return nil
			}
			if err := u.UsersBulkWithCustom(ctx, proc, u.includeLabels, ids...); err != nil {
				return err
			}
		}
	}
}
