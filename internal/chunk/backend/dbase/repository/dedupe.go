package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slackdump/v4/internal/chunk"
)

type MessageDedupeResult struct {
	MessagesRemoved int64
	ChunksRemoved   int64
}

type MessageDedupeRepository interface {
	CountDuplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error)
	CountPrunableMessageChunks(ctx context.Context, db *sqlx.DB) (int64, error)
	DeduplicateMessages(ctx context.Context, db *sqlx.DB) (MessageDedupeResult, error)
}

type messageDedupeRepository struct{}

func NewMessageDedupeRepository() MessageDedupeRepository {
	return messageDedupeRepository{}
}

func (messageDedupeRepository) CountDuplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error) {
	var count int64
	if err := db.QueryRowxContext(ctx, duplicateMessagesCountStmt).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r messageDedupeRepository) CountPrunableMessageChunks(ctx context.Context, db *sqlx.DB) (int64, error) {
	ids, err := r.prunableMessageChunkIDs(ctx, db)
	if err != nil {
		return 0, err
	}
	return int64(len(ids)), nil
}

func (r messageDedupeRepository) DeduplicateMessages(ctx context.Context, db *sqlx.DB) (MessageDedupeResult, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return MessageDedupeResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	chunkIDs, err := r.prunableMessageChunkIDs(ctx, tx)
	if err != nil {
		return MessageDedupeResult{}, err
	}

	result, err := tx.ExecContext(ctx, duplicateMessagesDeleteStmt)
	if err != nil {
		return MessageDedupeResult{}, fmt.Errorf("delete duplicate messages: %w", err)
	}
	messagesRemoved, err := result.RowsAffected()
	if err != nil {
		return MessageDedupeResult{}, fmt.Errorf("duplicate message rows affected: %w", err)
	}

	chunksRemoved, err := deleteChunksByID(ctx, tx, chunkIDs)
	if err != nil {
		return MessageDedupeResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return MessageDedupeResult{}, fmt.Errorf("commit: %w", err)
	}

	return MessageDedupeResult{
		MessagesRemoved: messagesRemoved,
		ChunksRemoved:   chunksRemoved,
	}, nil
}

func (messageDedupeRepository) prunableMessageChunkIDs(ctx context.Context, db sqlx.QueryerContext) ([]int64, error) {
	rows, err := db.QueryxContext(ctx, prunableMessageChunksStmt, chunk.CMessages, chunk.CThreadMessages)
	if err != nil {
		return nil, fmt.Errorf("query prunable message chunks: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan prunable message chunk: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func deleteChunksByID(ctx context.Context, tx *sqlx.Tx, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := strings.Join(placeholders(ids), ",")
	stmt := `DELETE FROM CHUNK WHERE ID IN (` + placeholders + `) AND TYPE_ID IN (?, ?)`
	args := make([]any, 0, len(ids)+2)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, chunk.CMessages, chunk.CThreadMessages)

	result, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return 0, fmt.Errorf("delete prunable chunks: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prunable chunk rows affected: %w", err)
	}
	return affected, nil
}

const duplicateMessagesCountStmt = `
WITH latest AS (
	SELECT ID, DATA, MAX(CHUNK_ID) AS KEEP_CHUNK_ID
	FROM MESSAGE
	GROUP BY ID, DATA
)
SELECT COUNT(1)
FROM MESSAGE M
JOIN latest L ON L.ID = M.ID AND L.DATA = M.DATA
WHERE M.CHUNK_ID < L.KEEP_CHUNK_ID`

const duplicateMessagesDeleteStmt = `
WITH latest AS (
	SELECT ID, DATA, MAX(CHUNK_ID) AS KEEP_CHUNK_ID
	FROM MESSAGE
	GROUP BY ID, DATA
)
DELETE FROM MESSAGE
WHERE (ID, CHUNK_ID) IN (
	SELECT M.ID, M.CHUNK_ID
	FROM MESSAGE M
	JOIN latest L ON L.ID = M.ID AND L.DATA = M.DATA
	WHERE M.CHUNK_ID < L.KEEP_CHUNK_ID
)`

const prunableMessageChunksStmt = `
WITH latest AS (
	SELECT ID, DATA, MAX(CHUNK_ID) AS KEEP_CHUNK_ID
	FROM MESSAGE
	GROUP BY ID, DATA
),
duplicates AS (
	SELECT M.ID, M.CHUNK_ID
	FROM MESSAGE M
	JOIN latest L ON L.ID = M.ID AND L.DATA = M.DATA
	WHERE M.CHUNK_ID < L.KEEP_CHUNK_ID
)
SELECT C.ID
FROM CHUNK C
LEFT JOIN MESSAGE M ON M.CHUNK_ID = C.ID
LEFT JOIN duplicates D ON D.ID = M.ID AND D.CHUNK_ID = M.CHUNK_ID
WHERE C.TYPE_ID IN (?, ?)
GROUP BY C.ID
HAVING COUNT(M.CHUNK_ID) > 0 AND COUNT(M.CHUNK_ID) = COUNT(D.CHUNK_ID)`
