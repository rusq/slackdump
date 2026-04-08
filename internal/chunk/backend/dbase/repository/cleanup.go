package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type CleanupResult struct {
	SessionsRemoved int64
	ChunksRemoved   int64
}

type CleanupRepository interface {
	CountUnfinishedSessions(ctx context.Context, db *sqlx.DB) (int64, error)
	CountUnfinishedChunks(ctx context.Context, db *sqlx.DB) (int64, error)
	CleanupUnfinishedSessions(ctx context.Context, db *sqlx.DB) (CleanupResult, error)
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

func (cleanupRepository) CleanupUnfinishedSessions(ctx context.Context, db *sqlx.DB) (CleanupResult, error) {
	sessionIDs, err := unfinishedSessionIDs(ctx, db)
	if err != nil {
		return CleanupResult{}, err
	}

	var result CleanupResult
	for _, sessionID := range sessionIDs {
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("begin transaction: %w", err)
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
