package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// Count indexed payload rows when choosing a batch boundary. A batch can exceed
// this target when the boundary falls in a group with many copies of a key.
const dedupeBatchRows = 400_000

// dedupeKey holds the indexed prefix of a dedupe identity. ID is an INTEGER
// for MESSAGE and TEXT for the other tables. CHANNEL_USER has two TEXT columns.
type dedupeKey struct {
	id   any
	user string
}

func (k dedupeKey) args(columns []string) []any {
	if len(columns) == 2 {
		return []any{k.id, k.user}
	}
	return []any{k.id}
}

type dedupeRange struct {
	columns []string
	lower   dedupeKey // inclusive
	upper   dedupeKey // exclusive
}

func (r dedupeRange) bounded() bool {
	return r.lower.id != nil || r.upper.id != nil
}

func dedupeKeyComparison(columns []string, alias string, op string) string {
	key := qualifyColumns(alias, columns)
	if len(columns) == 2 {
		return "(" + key + ") " + op + " (?, ?)"
	}
	return key + " " + op + " ?"
}

func (r dedupeRange) predicate(alias string) string {
	if !r.bounded() {
		return ""
	}
	if r.lower.id == nil {
		return dedupeKeyComparison(r.columns, alias, "<")
	}
	lower := dedupeKeyComparison(r.columns, alias, ">=")
	if r.upper.id == nil {
		return lower
	}
	return lower + " AND " + dedupeKeyComparison(r.columns, alias, "<")
}

func (r dedupeRange) appendArgs(args []any) []any {
	if r.lower.id != nil {
		args = append(args, r.lower.args(r.columns)...)
	}
	if r.upper.id != nil {
		args = append(args, r.upper.args(r.columns)...)
	}
	return args
}

func dedupeRangeIndex(table string) string {
	return "sqlite_autoindex_" + table + "_1"
}

func buildDedupeRangeCount(table string, bounds dedupeRange) string {
	stmt := "SELECT COUNT(*) FROM " + table + " T INDEXED BY " + dedupeRangeIndex(table)
	if predicate := bounds.predicate("T"); predicate != "" {
		stmt += " WHERE " + predicate
	}
	return stmt
}

func buildDedupeKeyProbe(table string, columns []string, op string) string {
	selected := qualifyColumns("T", columns)
	stmt := "SELECT " + selected + " FROM " + table + " T"
	if op != "" {
		stmt += " WHERE " + dedupeKeyComparison(columns, "T", op)
	}
	return stmt + " ORDER BY " + selected + " LIMIT 1 OFFSET ?"
}

// nextDedupeKey probes at most limit rows of the ordered index, then seeks to
// the first key after the probe. The exclusive boundary keeps every copy of
// the probe key together without counting the whole table.
func nextDedupeKey(ctx context.Context, tx *sqlx.Tx, table string, r dedupeRange, limit int) (dedupeKey, error) {
	var op string
	if r.lower.id != nil {
		op = ">="
	}
	stmt := buildDedupeKeyProbe(table, r.columns, op)
	probe, err := queryDedupeKey(ctx, tx, r.columns, stmt, append(r.appendArgs(nil), limit-1)...)
	if err != nil || probe.id == nil {
		return dedupeKey{}, err
	}

	// Bind the key directly here so SQLite can seek on both membership
	// columns. Comparing the tuple to a subquery only seeks on CHANNEL_ID.
	stmt = buildDedupeKeyProbe(table, r.columns, ">")
	return queryDedupeKey(ctx, tx, r.columns, stmt, append(probe.args(r.columns), 0)...)
}

func queryDedupeKey(ctx context.Context, tx *sqlx.Tx, columns []string, stmt string, args ...any) (dedupeKey, error) {
	var key dedupeKey
	dest := []any{&key.id}
	if len(columns) == 2 {
		dest = append(dest, &key.user)
	}
	err := tx.QueryRowxContext(ctx, tx.Rebind(stmt), args...).Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return dedupeKey{}, nil
	}
	return key, err
}

type dedupeCheckpoint struct {
	mode             string
	owner            string
	entity           entityType
	key              dedupeKey
	stats            DedupeStats
	rowsExamined     int64
	batchesCommitted int64
	countersScope    string
}

func readDedupeCheckpoint(ctx context.Context, tx *sqlx.Tx) (dedupeCheckpoint, error) {
	const stmt = `SELECT MODE, OWNER, ENTITY, KEY1, KEY2, MESSAGES, CANVAS_MESSAGES, USERS, CHANNELS, CHANNEL_USERS, FILES, CHUNKS,
		ROWS_EXAMINED, BATCHES_COMMITTED, COUNTERS_SCOPE
		FROM DEDUPE_CHECKPOINT WHERE ID = 1`
	var p dedupeCheckpoint
	err := tx.QueryRowxContext(ctx, stmt).Scan(&p.mode, &p.owner, &p.entity, &p.key.id, &p.key.user,
		&p.stats.Messages, &p.stats.CanvasMessages, &p.stats.Users, &p.stats.Channels, &p.stats.ChannelUsers, &p.stats.Files, &p.stats.Chunks,
		&p.rowsExamined, &p.batchesCommitted, &p.countersScope)
	if errors.Is(err, sql.ErrNoRows) {
		return dedupeCheckpoint{}, nil
	}
	return p, err
}

func (p dedupeCheckpoint) save(ctx context.Context, tx *sqlx.Tx) error {
	const stmt = `INSERT OR REPLACE INTO DEDUPE_CHECKPOINT
		(ID, MODE, OWNER, ENTITY, KEY1, KEY2, MESSAGES, CANVAS_MESSAGES, USERS, CHANNELS, CHANNEL_USERS, FILES, CHUNKS,
		ROWS_EXAMINED, BATCHES_COMMITTED, COUNTERS_SCOPE)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := tx.ExecContext(ctx, tx.Rebind(stmt), p.mode, p.owner, p.entity, p.key.id, p.key.user,
		p.stats.Messages, p.stats.CanvasMessages, p.stats.Users, p.stats.Channels, p.stats.ChannelUsers, p.stats.Files, p.stats.Chunks,
		p.rowsExamined, p.batchesCommitted, p.countersScope)
	return err
}

func (r dedupeRepository) deduplicateBatch(ctx context.Context, db *sqlx.DB, owner string, limit int) (dedupeCheckpoint, error) {
	started := time.Now()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return dedupeCheckpoint{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	progress, err := readDedupeCheckpoint(ctx, tx)
	if err != nil {
		return dedupeCheckpoint{}, fmt.Errorf("read dedupe checkpoint: %w", err)
	}
	// Check ownership in the transaction holding the same SQLite snapshot as
	// the duplicate selection and DELETE. A competing write after this read
	// makes a write from this transaction fail with SQLITE_BUSY.
	if owner != "" && progress.owner != owner {
		return dedupeCheckpoint{}, errors.New("dedupe checkpoint changed during execution; rerun dedupe")
	}
	hadCheckpoint := progress.owner != ""
	if progress.mode != r.messageMode.String() {
		progress = dedupeCheckpoint{mode: r.messageMode.String(), countersScope: "full_pass"}
	}
	priorStats := progress.stats
	priorExamined := progress.rowsExamined
	// The count is saved and reported only if this transaction commits.
	progress.batchesCommitted++

	entities := r.entities()
	tables := make([]string, 0, len(entities))
	var nextKey []any
	for i := int(progress.entity); i < len(entities); {
		entity := entities[i]
		tables = append(tables, entity.table)
		bounds := dedupeRange{columns: []string{"ID"}, lower: progress.key}
		if entity.etype == entChannelUsers {
			bounds.columns = entity.keyColumns
		}
		bounds.upper, err = nextDedupeKey(ctx, tx, entity.table, bounds, limit)
		if err != nil {
			return dedupeCheckpoint{}, fmt.Errorf("find next %s dedupe boundary: %w", entity.etype, err)
		}

		// Count the physical rows in the range before deleting any copies.
		// Ordinary and canvas messages share this count and the same bounds.
		var examined int64
		stmt := buildDedupeRangeCount(entity.table, bounds)
		if err := tx.GetContext(ctx, &examined, tx.Rebind(stmt), bounds.appendArgs(nil)...); err != nil {
			return dedupeCheckpoint{}, fmt.Errorf("count %s dedupe range: %w", entity.etype, err)
		}
		progress.rowsExamined += examined

		// Ordinary and canvas messages have separate duplicate groups and
		// removal counters.
		for i < len(entities) && entities[i].table == entity.table {
			if err := r.deduplicateRange(ctx, tx, entities[i], bounds, &progress.stats); err != nil {
				return dedupeCheckpoint{}, err
			}
			i++
		}
		if bounds.upper.id != nil {
			progress.entity = entity.etype
			progress.key = bounds.upper
			progress.owner = owner
			if progress.owner == "" {
				progress.owner = rand.Text()
			}
			if err := progress.save(ctx, tx); err != nil {
				return dedupeCheckpoint{}, fmt.Errorf("save dedupe checkpoint: %w", err)
			}
			nextKey = progress.key.args(bounds.columns)
			break
		}
		progress.key = dedupeKey{}
	}

	if progress.key.id == nil {
		// A small archive without an existing checkpoint never writes the
		// checkpoint table from dedupe itself.
		if hadCheckpoint {
			if _, err := tx.ExecContext(ctx, "DELETE FROM DEDUPE_CHECKPOINT WHERE ID = 1"); err != nil {
				return dedupeCheckpoint{}, fmt.Errorf("clear dedupe checkpoint: %w", err)
			}
		}
		progress.owner = ""
	}
	if err := tx.Commit(); err != nil {
		return dedupeCheckpoint{}, fmt.Errorf("commit dedupe: %w", err)
	}

	stats := progress.stats
	removedRows := stats.totalRows()
	attrs := []slog.Attr{
		slog.String("mode", progress.mode),
		slog.Any("tables", tables),
		slog.Int64("batch_rows_examined", progress.rowsExamined-priorExamined),
		slog.Int64("batch_rows_removed", removedRows-priorStats.totalRows()),
		slog.Int64("batch_chunks_removed", stats.Chunks-priorStats.Chunks),
		slog.Duration("batch_elapsed", time.Since(started)),
		slog.Int64("rows_examined_total", progress.rowsExamined),
		slog.Int64("batches_committed_total", progress.batchesCommitted),
		slog.String("counters_scope", progress.countersScope),
		slog.Int64("removed_rows_total", removedRows),
		slog.Int64("removed_chunks_total", stats.Chunks),
		slog.Bool("complete", nextKey == nil),
	}
	if nextKey != nil {
		attrs = append(attrs, slog.String("next_table", tables[len(tables)-1]), slog.Any("next_key", nextKey))
	}
	slog.LogAttrs(ctx, slog.LevelInfo, "dedupe batch committed", attrs...)
	return progress, nil
}
