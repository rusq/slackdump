package dbproc

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

var _ processor.Conversations = new(DBP)

type DBP struct {
	conn      *sqlx.DB
	sessionID int64
}

func (d *DBP) Messages(ctx context.Context, channelID string, numThreads int, isLast bool, messages []slack.Message) error {
	for i := range messages {
		slog.Info("  message", "ts", messages[i].Timestamp)
	}
	return nil
}

func (d *DBP) ThreadMessages(ctx context.Context, channelID string, parent slack.Message, threadOnly, isLast bool, replies []slack.Message) error {
	slog.Info("Discarding %d replies to %s", "n", len(replies), "parent_ts", parent.Timestamp)
	for i := range replies {
		slog.Info("  reply", "ts", replies[i].Timestamp)
	}
	return nil
}

func (d *DBP) Files(_ context.Context, ch *slack.Channel, parent slack.Message, files []slack.File) error {
	slog.Info("Discarding files", "n", len(files), "parent_ts", parent.Timestamp, "is_thread", parent.ThreadTimestamp != "")
	if parent.Timestamp == "" {
		runtime.Breakpoint()
	}
	for i := range files {
		slog.Info("  file", "id", files[i].ID)
	}
	return nil
}

func (d *DBP) ChannelInfo(_ context.Context, ch *slack.Channel, threadID string) error {
	sl := structures.SlackLink{Channel: ch.ID, ThreadTS: threadID}
	slog.Info("Discarding channel info", "channel_name", ch.Name, "slack_link", sl)
	return nil
}

func (d *DBP) ChannelUsers(_ context.Context, ch string, threadID string, u []string) error {
	sl := structures.SlackLink{Channel: ch, ThreadTS: threadID}
	slog.Info("Discarding channel users", "slack_link", sl, "users_len", len(u))
	return nil
}

func (d *DBP) Close() error {
	slog.Info("Discarder closing")
	return nil
}
