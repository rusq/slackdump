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
	"log/slog"
	"runtime"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4/internal/structures"
)

var _ Conversations = new(Printer)

type Printer struct{}

func (d *Printer) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	slog.Info("Discarding messages", "n", len(messages))
	for i := range messages {
		slog.Info("  message", "ts", messages[i].Timestamp)
	}
	return nil
}

func (d *Printer) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	slog.Info("Discarding %d replies to %s", "n", len(replies), "parent_ts", parent.Timestamp)
	for i := range replies {
		slog.Info("  reply", "ts", replies[i].Timestamp)
	}
	return nil
}

func (d *Printer) Files(_ context.Context, ch *slack.Channel, parent slack.Message, files []slack.File) error {
	slog.Info("Discarding files", "n", len(files), "parent_ts", parent.Timestamp, "is_thread", parent.ThreadTimestamp != "")
	if parent.Timestamp == "" {
		runtime.Breakpoint()
	}
	for i := range files {
		slog.Info("  file", "id", files[i].ID)
	}
	return nil
}

func (d *Printer) ChannelInfo(_ context.Context, ch *slack.Channel, threadID string) error {
	sl := structures.SlackLink{Channel: ch.ID, ThreadTS: threadID}
	slog.Info("Discarding channel info", "channel_name", ch.Name, "slack_link", sl)
	return nil
}

func (d *Printer) ChannelUsers(_ context.Context, ch string, threadID string, u []string) error {
	sl := structures.SlackLink{Channel: ch, ThreadTS: threadID}
	slog.Info("Discarding channel users", "slack_link", sl, "users_len", len(u))
	return nil
}

func (d *Printer) Close() error {
	slog.Info("Discarder closing")
	return nil
}
