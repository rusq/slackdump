package dbproc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/dbproc/repository"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/processor"
)

var _ processor.Conversations = new(DBP)

type DBP struct {
	conn      *sqlx.DB
	sessionID int64
}

type Parameters struct {
	FromTS         *time.Time
	ToTS           *time.Time
	FilesEnabled   bool
	AvatarsEnabled bool
	Mode           string
	Args           string
}

func New(ctx context.Context, conn *sqlx.DB, p Parameters) (*DBP, error) {
	if err := repository.Migrate(ctx, conn.DB); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	sr := repository.NewSessionRepository()
	id, err := sr.Insert(ctx, conn, &repository.Session{
		CreatedAt:      time.Time{},
		ParentID:       new(int64),
		FromTS:         p.FromTS,
		ToTS:           p.ToTS,
		FilesEnabled:   p.FilesEnabled,
		AvatarsEnabled: p.AvatarsEnabled,
		Mode:           p.Mode,
		Args:           p.Args,
	})
	if err != nil {
		return nil, fmt.Errorf("new: %w", err)
	}
	// enable foreign keys
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("PRAGMA foreign_keys: %w", err)
	}
	return &DBP{conn: conn, sessionID: id}, nil
}

func (d *DBP) Close() error {
	sr := repository.NewSessionRepository()
	if n, err := sr.Finish(context.Background(), d.conn, d.sessionID); err != nil {
		return fmt.Errorf("finish: %w", err)
	} else if n == 0 {
		return errors.New("finish: no session found")
	}
	return nil
}

func (d *DBP) Encode(ctx context.Context, ch any) error {
	c, ok := ch.(chunk.Chunk)
	if !ok {
		return fmt.Errorf("invalid chunk type %T", ch)
	}
	if _, err := d.InsertChunk(ctx, c); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
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
