package dbase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"
)

// DBP is the database processor.
type DBP struct {
	mu        sync.RWMutex
	conn      *sqlx.DB
	sessionID int64
	closed    atomic.Bool

	mr repository.MessageRepository
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

	return &DBP{
		conn:      conn,
		sessionID: id,
		mr:        repository.NewMessageRepository(),
	}, nil
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
	if swapped := d.closed.CompareAndSwap(false, true); !swapped {
		return nil
	}
	sr := repository.NewSessionRepository()
	if n, err := sr.Finalise(context.Background(), d.conn, d.sessionID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("finish: %w", err)
	} else if n == 0 {
		return errors.New("finish: no session found")
	}
	return nil
}

// Encode inserts the chunk into the database.
func (d *DBP) Encode(ctx context.Context, ch *chunk.Chunk) error {
	if _, err := d.InsertChunk(ctx, ch); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// IsFinalised returns true if the channel messages have been processed (there
// are no unfinished threads).
func (d *DBP) IsComplete(ctx context.Context, channelID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.mr.CountUnfinished(ctx, d.conn, d.sessionID, channelID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("countUnfinished: %w", err)
	}
	return n <= 0, nil
}

// Source returns the connection that can be used safely as a source.
func (d *DBP) Source() *Source {
	return &Source{
		conn:     d.conn,
		canClose: false,
	}
}
