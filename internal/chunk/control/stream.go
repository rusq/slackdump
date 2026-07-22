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
	"errors"
	"log/slog"

	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
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

const userNotFound = "user_not_found"

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
			if err := u.UsersBulkWithCustomErr(ctx, proc, u.includeLabels, ids, func(err error) bool {
				if structures.IsSlackResponseError(err, userNotFound) {
					var lookupErr interface{ UserID() string }
					if errors.As(err, &lookupErr) {
						slog.WarnContext(ctx, "skipping unresolved message author", "user_id", lookupErr.UserID())
					} else {
						slog.WarnContext(ctx, "skipping unresolved message author", "error", err)
					}
					return false
				}
				return true
			}); err != nil {
				return err
			}
		}
	}
}
