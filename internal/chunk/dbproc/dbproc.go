package dbproc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

// DBP is the database processor.
type DBP struct {
	mu        sync.RWMutex
	conn      *sqlx.DB
	sessionID int64
}

func (d *DBP) String() string {
	return fmt.Sprintf("<DBP:%d>", d.sessionID)
}

var _ chunk.Encoder = (*DBP)(nil)

// SessionInfo is the information about the session to be logged in the
// database.
type SessionInfo struct {
	FromTS         *time.Time
	ToTS           *time.Time
	FilesEnabled   bool
	AvatarsEnabled bool
	Mode           string
	Args           string
}

var dbInitCommands = []string{
	"PRAGMA journal_mode=WAL",   // enable WAL mode
	"PRAGMA synchronous=NORMAL", // enable synchronous mode
	"PRAGMA foreign_keys=ON",    // enable foreign keys
}

type options struct {
	verbose bool
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(*options)

func WithVerbose(v bool) Option {
	return func(o *options) {
		o.verbose = v
	}
}

// New return the new database processor.
func New(ctx context.Context, conn *sqlx.DB, p SessionInfo, opts ...Option) (*DBP, error) {
	var options options
	options.apply(opts...)

	if err := initDB(ctx, conn); err != nil {
		return nil, fmt.Errorf("new: %w", err)
	}

	if err := repository.Migrate(ctx, conn.DB, options.verbose); err != nil {
		return nil, fmt.Errorf("new: %w", err)
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

	return &DBP{conn: conn, sessionID: id}, nil
}

// initDB runs the initialisation commands on the database.
func initDB(ctx context.Context, conn *sqlx.DB) error {
	for _, q := range dbInitCommands {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("initDB: %w", err)
		}
	}
	return nil
}

// Close finalises the session, marking it as finished. It is advised to check
// the error value.
func (d *DBP) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	sr := repository.NewSessionRepository()
	if n, err := sr.Finalise(context.Background(), d.conn, d.sessionID); err != nil {
		return fmt.Errorf("finish: %w", err)
	} else if n == 0 {
		return errors.New("finish: no session found")
	}
	return nil
}

// Encode inserts the chunk into the database.
func (d *DBP) Encode(ctx context.Context, ch chunk.Chunk) error {
	if n, err := d.InsertChunk(ctx, ch); err != nil {
		return fmt.Errorf("encode: %w", err)
	} else {
		slog.DebugContext(ctx, "inserted chunk", "id", n, "type", ch.Type)
	}
	return nil
}

// IsFinalised returns true if the channel messages have been processed (there
// are no unfinished threads).
func (d *DBP) IsComplete(ctx context.Context, channelID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	mr := repository.NewMessageRepository()
	n, err := mr.CountUnfinished(ctx, d.conn, d.sessionID, channelID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("countUnfinished: %w", err)
	}
	return n <= 0, nil
}

// Finalise is a no-op for the database processor.
func (d *DBP) Finalise(ctx context.Context, channelID string) error {
	// noop
	return nil
}

// Source returns the connection that can be used safely as a source.
func (d *DBP) Source() *Source {
	return &Source{
		conn:     d.conn,
		canClose: false,
	}
}
