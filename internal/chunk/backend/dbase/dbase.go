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

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

// DBP is the database processor.
type DBP struct {
	mu        sync.RWMutex
	conn      *sqlx.DB
	sessionID int64
	closed    atomic.Bool

	mr   repository.MessageRepository
	opts options
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
	onlyNewOrChangedUsers bool
	verbose               bool
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

func WithOnlyNewOrChangedUsers(v bool) Option {
	return func(o *options) {
		o.onlyNewOrChangedUsers = v
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
		opts:      options,
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

// IsComplete returns true if the channel messages have been processed (there
// are no unfinished threads, and all messages were received).
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

// IsCompleteThread checks that thread with channelID and threadID is complete for
// thread-only archives.  It returns true if there are no unfinished parts of the
// thread.  It returns false if the thread is not found.  It will return false
// on non-thread-only archives.
func (d *DBP) IsCompleteThread(ctx context.Context, channelID, threadID string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.mr.CountThreadOnlyParts(ctx, d.conn, d.sessionID, channelID, threadID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("countUnfinished: %w", err)
	}
	// note that count thread only parts returns non-zero for completed threads,
	// so the check is reversed.
	return n > 0, nil
}

// Source returns the connection that can be used safely as a source.
func (d *DBP) Source() *Source {
	return &Source{
		conn:     d.conn,
		canClose: false,
	}
}
