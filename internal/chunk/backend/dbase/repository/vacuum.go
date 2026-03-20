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

package repository

// Each vacuum operation (deduplicate users, deduplicate messages, prune chunks)
// runs in its own separate transaction. This is intentional - if one fails, we
// want to preserve the partial progress of the other operations. For example,
// if deduplicating messages succeeds but pruning chunks fails, the user can
// re-run just the chunk pruning without having to redo the message deduplication.

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slackdump/v4/internal/chunk"
)

// VacuumRepository provides database vacuum operations.
type VacuumRepository interface {
	PruneUnreferencedChunks(ctx context.Context, db *sqlx.DB) (int64, error)
	CountUnreferencedChunks(ctx context.Context, db *sqlx.DB) (int64, error)
	DeduplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error)
	CountDuplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error)
	DeduplicateUsers(ctx context.Context, db *sqlx.DB) (int64, error)
	CountDuplicateUsers(ctx context.Context, db *sqlx.DB) (int64, error)
	DeduplicateFiles(ctx context.Context, db *sqlx.DB) (int64, error)
	CountDuplicateFiles(ctx context.Context, db *sqlx.DB) (int64, error)
}

type vacuumRepository struct{}

func NewVacuumRepository() VacuumRepository {
	return vacuumRepository{}
}

// entityInfo maps chunk types to their corresponding database tables.
type entityInfo struct {
	types []chunk.ChunkType // chunk types that map to this entity
	table string            // database table name
}

// entityMappings defines the relationship between chunk types and database tables.
var entityMappings = []entityInfo{
	{types: []chunk.ChunkType{chunk.CMessages, chunk.CThreadMessages}, table: "MESSAGE"},
	{types: []chunk.ChunkType{chunk.CFiles}, table: "FILE"},
	{types: []chunk.ChunkType{chunk.CUsers}, table: "S_USER"},
}

// countUnreferencedChunksFn returns a function that counts chunks with no entries
// in the associated table (MESSAGE, FILE, or S_USER).
func countUnreferencedChunksFn(info entityInfo) func(ctx context.Context, db *sqlx.DB) (int64, error) {
	stmt := fmt.Sprintf(`
		SELECT COUNT(1) FROM CHUNK
		WHERE TYPE_ID IN (%s)
		  AND ID NOT IN (SELECT DISTINCT CHUNK_ID FROM %s)`,
		chunkTypePlaceholders(len(info.types)), info.table)

	return func(ctx context.Context, db *sqlx.DB) (int64, error) {
		var count int64
		err := db.QueryRowxContext(ctx, stmt, chunkTypesToArgs(info.types)...).Scan(&count)
		return count, err
	}
}

// deleteUnreferencedChunksFn returns a function that deletes chunks with no entries
// in the associated table (MESSAGE, FILE, or S_USER).
func deleteUnreferencedChunksFn(info entityInfo) func(ctx context.Context, tx *sqlx.Tx) (int64, error) {
	stmt := fmt.Sprintf(`
		DELETE FROM CHUNK
		WHERE TYPE_ID IN (%s)
		  AND ID NOT IN (SELECT DISTINCT CHUNK_ID FROM %s)`,
		chunkTypePlaceholders(len(info.types)), info.table)

	return func(ctx context.Context, tx *sqlx.Tx) (int64, error) {
		result, err := tx.ExecContext(ctx, stmt, chunkTypesToArgs(info.types)...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}
}

// countDuplicatesFn returns a function that counts duplicate rows in the given table.
// Duplicates are rows with the same ID and DATA but different CHUNK_ID values.
func countDuplicatesFn(table string) func(ctx context.Context, db *sqlx.DB) (int64, error) {
	stmt := fmt.Sprintf(`
		WITH earliest AS (
			SELECT ID, MIN(CHUNK_ID) as CHUNK_ID, DATA
			FROM %s GROUP BY ID
		)
		SELECT COUNT(1) FROM %s T
		JOIN earliest E ON T.ID = E.ID
		WHERE T.CHUNK_ID > E.CHUNK_ID AND T.DATA = E.DATA`, table, table)

	return func(ctx context.Context, db *sqlx.DB) (int64, error) {
		var count int64
		err := db.QueryRowxContext(ctx, stmt).Scan(&count)
		return count, err
	}
}

// deleteDuplicatesFn returns a function that removes duplicate rows from the given table.
// It keeps only the earliest occurrence (by CHUNK_ID) for each ID where the DATA matches.
func deleteDuplicatesFn(table string) func(ctx context.Context, db *sqlx.DB) (int64, error) {
	stmt := fmt.Sprintf(`
		WITH earliest AS (
			SELECT ID, MIN(CHUNK_ID) as CHUNK_ID, DATA
			FROM %s GROUP BY ID
		)
		DELETE FROM %s
		WHERE (ID, CHUNK_ID) IN (
			SELECT T.ID, T.CHUNK_ID FROM %s T
			JOIN earliest E ON T.ID = E.ID
			WHERE T.CHUNK_ID > E.CHUNK_ID AND T.DATA = E.DATA
		)`, table, table, table)

	return func(ctx context.Context, db *sqlx.DB) (int64, error) {
		tx, err := db.BeginTxx(ctx, nil)
		if err != nil {
			return 0, fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback()

		result, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return 0, err
		}

		if err := tx.Commit(); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}

		return result.RowsAffected()
	}
}

func (r vacuumRepository) PruneUnreferencedChunks(ctx context.Context, db *sqlx.DB) (int64, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var totalDeleted int64
	for _, info := range entityMappings {
		deleted, err := deleteUnreferencedChunksFn(info)(ctx, tx)
		if err != nil {
			return totalDeleted, err
		}
		totalDeleted += deleted
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return totalDeleted, nil
}

func (r vacuumRepository) CountUnreferencedChunks(ctx context.Context, db *sqlx.DB) (int64, error) {
	var totalCount int64
	for _, info := range entityMappings {
		count, err := countUnreferencedChunksFn(info)(ctx, db)
		if err != nil {
			return totalCount, err
		}
		totalCount += count
	}
	return totalCount, nil
}

func (r vacuumRepository) DeduplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error) {
	return deleteDuplicatesFn("MESSAGE")(ctx, db)
}

func (r vacuumRepository) CountDuplicateMessages(ctx context.Context, db *sqlx.DB) (int64, error) {
	return countDuplicatesFn("MESSAGE")(ctx, db)
}

func (r vacuumRepository) DeduplicateUsers(ctx context.Context, db *sqlx.DB) (int64, error) {
	return deleteDuplicatesFn("S_USER")(ctx, db)
}

func (r vacuumRepository) CountDuplicateUsers(ctx context.Context, db *sqlx.DB) (int64, error) {
	return countDuplicatesFn("S_USER")(ctx, db)
}

func (r vacuumRepository) DeduplicateFiles(ctx context.Context, db *sqlx.DB) (int64, error) {
	return deleteDuplicatesFn("FILE")(ctx, db)
}

func (r vacuumRepository) CountDuplicateFiles(ctx context.Context, db *sqlx.DB) (int64, error) {
	return countDuplicatesFn("FILE")(ctx, db)
}

// chunkTypePlaceholders returns a comma-separated list of placeholders (e.g., "?,?,?").
func chunkTypePlaceholders(n int) string {
	if n <= 0 {
		return ""
	}
	placeholders := make([]byte, n*2-1)
	for i := 0; i < n; i++ {
		if i > 0 {
			placeholders[i*2-1] = ','
		}
		placeholders[i*2] = '?'
	}
	return string(placeholders)
}

// chunkTypesToArgs converts chunk types to a slice of interface{} for SQL args.
func chunkTypesToArgs(types []chunk.ChunkType) []any {
	args := make([]any, len(types))
	for i, t := range types {
		args[i] = t
	}
	return args
}
