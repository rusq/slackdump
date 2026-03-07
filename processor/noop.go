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
