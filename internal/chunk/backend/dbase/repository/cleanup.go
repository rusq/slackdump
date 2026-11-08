package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// PayloadTables lists the tables that hold chunk payload rows, i.e. tables
// with a CHUNK_ID column referencing CHUNK.ID.
var PayloadTables = []string{
	"MESSAGE",
	"CHANNEL",
	"FILE",
	"WORKSPACE",
	"S_USER",
	"CHANNEL_USER",
	"SEARCH_MESSAGE",
	"SEARCH_FILE",
}

// TableCounts maps a payload table name to a row count.
type TableCounts map[string]int64

// Total returns the sum of all counts.
func (tc TableCounts) Total() int64 {
	var n int64
	for _, v := range tc {
		n += v
	}
	return n
}

func (tc TableCounts) add(other TableCounts) {
	for k, v := range other {
		tc[k] += v
	}
}

type CleanupResult struct {
	SessionsRemoved int64
	ChunksRemoved   int64
	// PayloadRemoved is the number of payload rows removed per table, that
	// belonged to chunks of unfinished sessions.
	PayloadRemoved TableCounts
}

type CleanupRepository interface {
	CountUnfinishedSessions(ctx context.Context, db *sqlx.DB) (int64, error)
	CountUnfinishedChunks(ctx context.Context, db *sqlx.DB) (int64, error)
	// CountUnfinishedPayload returns the number of payload rows per table
	// that belong to chunks of unfinished sessions.
	CountUnfinishedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error)
	CleanupUnfinishedSessions(ctx context.Context, db *sqlx.DB) (CleanupResult, error)
	// CountOrphanedPayload returns the number of payload rows per table
	// whose CHUNK_ID does not reference an existing chunk.
	CountOrphanedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error)
	// DeleteOrphanedPayload deletes payload rows whose CHUNK_ID does not
	// reference an existing chunk, and returns the number of rows removed
	// per table.
	DeleteOrphanedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error)
}

type cleanupRepository struct{}

func NewCleanupRepository() CleanupRepository {
	return cleanupRepository{}
}

func (cleanupRepository) CountUnfinishedSessions(ctx context.Context, db *sqlx.DB) (int64, error) {
	const stmt = `SELECT COUNT(1) FROM SESSION WHERE FINISHED = ?`
	var count int64
	if err := db.QueryRowxContext(ctx, stmt, false).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (cleanupRepository) CountUnfinishedChunks(ctx context.Context, db *sqlx.DB) (int64, error) {
	const stmt = `
		SELECT COUNT(1)
		FROM CHUNK
		WHERE SESSION_ID IN (
			SELECT ID FROM SESSION WHERE FINISHED = ?
		)`
	var count int64
	if err := db.QueryRowxContext(ctx, stmt, false).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (cleanupRepository) CountUnfinishedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error) {
	return countPayload(ctx, db, `CHUNK_ID IN (SELECT C.ID FROM CHUNK C JOIN SESSION S ON S.ID = C.SESSION_ID WHERE S.FINISHED = ?)`, false)
}

// orphanPredicate is a WHERE clause matching payload rows of table T that
// reference a non-existent chunk.
const orphanPredicate = `NOT EXISTS (SELECT 1 FROM CHUNK C WHERE C.ID = %[1]s.CHUNK_ID)`

func (cleanupRepository) CountOrphanedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error) {
	return countPayload(ctx, db, orphanPredicate)
}

func (cleanupRepository) DeleteOrphanedPayload(ctx context.Context, db *sqlx.DB) (TableCounts, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	removed, err := deletePayload(ctx, tx, orphanPredicate)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return removed, nil
}

// countPayload counts rows matching the predicate in each payload table.  The
// predicate may reference the table name with %[1]s.
func countPayload(ctx context.Context, db sqlx.QueryerContext, predicate string, args ...any) (TableCounts, error) {
	counts := make(TableCounts, len(PayloadTables))
	for _, table := range PayloadTables {
		stmt := fmt.Sprintf("SELECT COUNT(1) FROM %[1]s WHERE "+predicate, table)
		var n int64
		if err := db.QueryRowxContext(ctx, stmt, args...).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s rows: %w", table, err)
		}
		counts[table] = n
	}
	return counts, nil
}

// deletePayload deletes rows matching the predicate from each payload table.
// The predicate may reference the table name with %[1]s.
func deletePayload(ctx context.Context, tx *sqlx.Tx, predicate string, args ...any) (TableCounts, error) {
	removed := make(TableCounts, len(PayloadTables))
	for _, table := range PayloadTables {
		stmt := fmt.Sprintf("DELETE FROM %[1]s WHERE "+predicate, table)
		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return nil, fmt.Errorf("delete %s rows: %w", table, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("%s rows affected: %w", table, err)
		}
		removed[table] = n
	}
	return removed, nil
}

// deleteSessionPayload deletes payload rows that belong to chunks of the
// given session.  It must be called before deleting the session chunks, as
// cascading deletes cannot be relied upon: foreign keys may be disabled on
// the connection.
func deleteSessionPayload(ctx context.Context, tx *sqlx.Tx, sessionID int64) (TableCounts, error) {
	removed, err := deletePayload(ctx, tx, `CHUNK_ID IN (SELECT ID FROM CHUNK WHERE SESSION_ID = ?)`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session %d: %w", sessionID, err)
	}
	return removed, nil
}

func (cleanupRepository) CleanupUnfinishedSessions(ctx context.Context, db *sqlx.DB) (CleanupResult, error) {
	sessionIDs, err := unfinishedSessionIDs(ctx, db)
	if err != nil {
		return CleanupResult{}, err
	}

	result := CleanupResult{PayloadRemoved: make(TableCounts, len(PayloadTables))}
	for _, sessionID := range sessionIDs {
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("begin transaction: %w", err)
		}

		payloadRemoved, err := deleteSessionPayload(ctx, tx, sessionID)
		if err != nil {
			tx.Rollback()
			return result, err
		}

		chunksRemoved, err := deleteSessionChunks(ctx, tx, sessionID)
		if err != nil {
			tx.Rollback()
			return result, err
		}

		sessionsRemoved, err := deleteUnfinishedSession(ctx, tx, sessionID)
		if err != nil {
			tx.Rollback()
			return result, err
		}

		if err := tx.Commit(); err != nil {
			return result, fmt.Errorf("commit: %w", err)
		}

		result.PayloadRemoved.add(payloadRemoved)
		result.ChunksRemoved += chunksRemoved
		result.SessionsRemoved += sessionsRemoved
	}
	return result, nil
}

func unfinishedSessionIDs(ctx context.Context, db sqlx.QueryerContext) ([]int64, error) {
	const stmt = `SELECT ID FROM SESSION WHERE FINISHED = ? ORDER BY ID`
	rows, err := db.QueryxContext(ctx, stmt, false)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func deleteSessionChunks(ctx context.Context, tx *sqlx.Tx, sessionID int64) (int64, error) {
	const stmt = `DELETE FROM CHUNK WHERE SESSION_ID = ?`
	result, err := tx.ExecContext(ctx, stmt, sessionID)
	if err != nil {
		return 0, fmt.Errorf("delete chunks for session %d: %w", sessionID, err)
	}
	return result.RowsAffected()
}

func deleteUnfinishedSession(ctx context.Context, tx *sqlx.Tx, sessionID int64) (int64, error) {
	const stmt = `DELETE FROM SESSION WHERE ID = ? AND FINISHED = ?`
	result, err := tx.ExecContext(ctx, stmt, sessionID, false)
	if err != nil {
		return 0, fmt.Errorf("delete session %d: %w", sessionID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected for session %d: %w", sessionID, err)
	}
	if affected == 0 {
		return 0, fmt.Errorf("session %d: not deleted (may have been finalized concurrently)", sessionID)
	}
	return affected, nil
}
