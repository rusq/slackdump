package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

type DedupeCounts struct {
	Messages     int64
	Users        int64
	Channels     int64
	ChannelUsers int64
	Files        int64
	Chunks       int64
}

type DedupeResult struct {
	MessagesRemoved     int64
	UsersRemoved        int64
	ChannelsRemoved     int64
	ChannelUsersRemoved int64
	FilesRemoved        int64
	ChunksRemoved       int64
}

type DedupeRepository interface {
	Preview(ctx context.Context, db *sqlx.DB) (DedupeCounts, error)
	Deduplicate(ctx context.Context, db *sqlx.DB) (DedupeResult, error)
}

type dedupeRepository struct{}

type dedupeMode int

const (
	dedupeByData dedupeMode = iota
	dedupeByKey
)

type dedupeEntity struct {
	name       string
	table      string
	keyColumns []string
	chunkTypes []chunk.ChunkType
	mode       dedupeMode
}

var dedupeEntities = []dedupeEntity{
	{
		name:       "messages",
		table:      "MESSAGE",
		keyColumns: []string{"ID"},
		chunkTypes: []chunk.ChunkType{chunk.CMessages, chunk.CThreadMessages},
		mode:       dedupeByData,
	},
	{
		name:       "users",
		table:      "S_USER",
		keyColumns: []string{"ID"},
		chunkTypes: []chunk.ChunkType{chunk.CUsers},
		mode:       dedupeByData,
	},
	{
		name:       "channels",
		table:      "CHANNEL",
		keyColumns: []string{"ID"},
		chunkTypes: []chunk.ChunkType{chunk.CChannels, chunk.CChannelInfo},
		mode:       dedupeByData,
	},
	{
		name:       "channel users",
		table:      "CHANNEL_USER",
		keyColumns: []string{"CHANNEL_ID", "USER_ID"},
		chunkTypes: []chunk.ChunkType{chunk.CChannelUsers},
		mode:       dedupeByKey,
	},
	{
		name:       "files",
		table:      "FILE",
		keyColumns: []string{"ID", "CHANNEL_ID", "MESSAGE_ID", "THREAD_ID"},
		chunkTypes: []chunk.ChunkType{chunk.CFiles},
		mode:       dedupeByData,
	},
}

func NewDedupeRepository() DedupeRepository {
	return dedupeRepository{}
}

func (r dedupeRepository) Preview(ctx context.Context, db *sqlx.DB) (DedupeCounts, error) {
	var counts DedupeCounts
	for _, entity := range dedupeEntities {
		n, err := r.countDuplicates(ctx, db, entity)
		if err != nil {
			return DedupeCounts{}, fmt.Errorf("count duplicate %s: %w", entity.name, err)
		}
		assignCount(&counts, entity.name, n)

		chunks, err := r.prunableChunkIDs(ctx, db, entity)
		if err != nil {
			return DedupeCounts{}, fmt.Errorf("count prunable %s chunks: %w", entity.name, err)
		}
		counts.Chunks += int64(len(chunks))
	}
	return counts, nil
}

func (r dedupeRepository) Deduplicate(ctx context.Context, db *sqlx.DB) (DedupeResult, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return DedupeResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var result DedupeResult
	for _, entity := range dedupeEntities {
		chunkIDs, err := r.prunableChunkIDs(ctx, tx, entity)
		if err != nil {
			return DedupeResult{}, fmt.Errorf("query prunable %s chunks: %w", entity.name, err)
		}

		rowsRemoved, err := deleteDuplicates(ctx, tx, entity)
		if err != nil {
			return DedupeResult{}, fmt.Errorf("delete duplicate %s: %w", entity.name, err)
		}
		assignRemoved(&result, entity.name, rowsRemoved)

		chunksRemoved, err := deleteChunksByID(ctx, tx, chunkIDs, entity.chunkTypes)
		if err != nil {
			return DedupeResult{}, fmt.Errorf("delete prunable %s chunks: %w", entity.name, err)
		}
		result.ChunksRemoved += chunksRemoved
	}

	if err := tx.Commit(); err != nil {
		return DedupeResult{}, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

func assignCount(counts *DedupeCounts, entityName string, n int64) {
	switch entityName {
	case "messages":
		counts.Messages = n
	case "users":
		counts.Users = n
	case "channels":
		counts.Channels = n
	case "channel users":
		counts.ChannelUsers = n
	case "files":
		counts.Files = n
	}
}

func assignRemoved(result *DedupeResult, entityName string, n int64) {
	switch entityName {
	case "messages":
		result.MessagesRemoved = n
	case "users":
		result.UsersRemoved = n
	case "channels":
		result.ChannelsRemoved = n
	case "channel users":
		result.ChannelUsersRemoved = n
	case "files":
		result.FilesRemoved = n
	}
}

func (r dedupeRepository) countDuplicates(ctx context.Context, db sqlx.QueryerContext, entity dedupeEntity) (int64, error) {
	stmt := withDuplicateRows(entity, `SELECT COUNT(1) FROM duplicates`)
	rows, err := queryxContext(ctx, db, stmt, chunkTypeArgs(entity.chunkTypes)...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	if !rows.Next() {
		return 0, rows.Err()
	}
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r dedupeRepository) prunableChunkIDs(ctx context.Context, db sqlx.QueryerContext, entity dedupeEntity) ([]int64, error) {
	stmt := withDuplicateRows(entity, buildPrunableChunksSelect(entity))
	args := append(chunkTypeArgs(entity.chunkTypes), chunkTypeArgs(entity.chunkTypes)...)
	rows, err := queryxContext(ctx, db, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan prunable %s chunk: %w", entity.name, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func deleteDuplicates(ctx context.Context, tx *sqlx.Tx, entity dedupeEntity) (int64, error) {
	stmt := withDuplicateRows(entity, buildDeleteDuplicatesStmt(entity))
	result, err := tx.ExecContext(ctx, tx.Rebind(stmt), chunkTypeArgs(entity.chunkTypes)...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("duplicate %s rows affected: %w", entity.name, err)
	}
	return affected, nil
}

func deleteChunksByID(ctx context.Context, tx *sqlx.Tx, ids []int64, chunkTypes []chunk.ChunkType) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// batchSize limits the number of chunk IDs per DELETE statement to stay
	// within SQLite's SQLITE_MAX_VARIABLE_NUMBER limit.
	const batchSize = 10000
	typeArgs := chunkTypeArgs(chunkTypes)
	typePlaceholders := strings.Join(placeholders(chunkTypes), ",")

	var totalAffected int64
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]

		var buf strings.Builder
		buf.WriteString("DELETE FROM CHUNK WHERE ID IN (")
		buf.WriteString(strings.Join(placeholders(batch), ","))
		buf.WriteString(") AND TYPE_ID IN (")
		buf.WriteString(typePlaceholders)
		buf.WriteString(")")

		args := make([]any, 0, len(batch)+len(typeArgs))
		for _, id := range batch {
			args = append(args, id)
		}
		args = append(args, typeArgs...)

		result, err := tx.ExecContext(ctx, tx.Rebind(buf.String()), args...)
		if err != nil {
			return totalAffected, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return totalAffected, fmt.Errorf("prunable chunk rows affected: %w", err)
		}
		totalAffected += affected
	}
	return totalAffected, nil
}

func buildPrunableChunksSelect(entity dedupeEntity) string {
	var buf strings.Builder
	buf.WriteString("SELECT C.ID\n")
	buf.WriteString("FROM CHUNK C\n")
	buf.WriteString("LEFT JOIN ")
	buf.WriteString(entity.table)
	buf.WriteString(" T ON T.CHUNK_ID = C.ID\n")
	buf.WriteString("LEFT JOIN duplicates D ON D.CHUNK_ID = T.CHUNK_ID")
	if len(entity.keyColumns) > 0 {
		buf.WriteString(" AND ")
		buf.WriteString(joinOnColumns("D", "T", entity.keyColumns))
	}
	buf.WriteString("\nWHERE C.TYPE_ID IN (")
	buf.WriteString(strings.Join(placeholders(entity.chunkTypes), ","))
	buf.WriteString(")\nGROUP BY C.ID\nHAVING COUNT(T.CHUNK_ID) > 0 AND COUNT(T.CHUNK_ID) = COUNT(D.CHUNK_ID)")
	return buf.String()
}

func buildDeleteDuplicatesStmt(entity dedupeEntity) string {
	var buf strings.Builder
	buf.WriteString("DELETE FROM ")
	buf.WriteString(entity.table)
	buf.WriteString(" AS T\nWHERE EXISTS (\nSELECT 1 FROM duplicates D WHERE D.CHUNK_ID = T.CHUNK_ID")
	if len(entity.keyColumns) > 0 {
		buf.WriteString(" AND ")
		buf.WriteString(joinOnColumns("D", "T", entity.keyColumns))
	}
	buf.WriteString("\n)")
	return buf.String()
}

func withDuplicateRows(entity dedupeEntity, final string) string {
	var buf strings.Builder
	buf.WriteString("WITH latest AS (\n")
	buf.WriteString("SELECT ")
	buf.WriteString(selectAliasedColumns("T", entity.keyColumns))
	if entity.mode == dedupeByData {
		buf.WriteString(", T.DATA AS DATA")
	}
	buf.WriteString(", MAX(T.CHUNK_ID) AS KEEP_CHUNK_ID\n")
	buf.WriteString("FROM ")
	buf.WriteString(entity.table)
	buf.WriteString(" T\nJOIN CHUNK C ON C.ID = T.CHUNK_ID\n")
	buf.WriteString("WHERE C.TYPE_ID IN (")
	buf.WriteString(strings.Join(placeholders(entity.chunkTypes), ","))
	buf.WriteString(")\nGROUP BY ")
	buf.WriteString(qualifyColumns("T", entity.keyColumns))
	if entity.mode == dedupeByData {
		buf.WriteString(", T.DATA")
	}
	buf.WriteString("\n),\nduplicates AS (\n")
	buf.WriteString("SELECT T.")
	buf.WriteString(strings.Join(entity.keyColumns, ", T."))
	buf.WriteString(", T.CHUNK_ID\n")
	buf.WriteString("FROM ")
	buf.WriteString(entity.table)
	buf.WriteString(" T\nJOIN latest L ON ")
	buf.WriteString(joinOnColumns("T", "L", entity.keyColumns))
	if entity.mode == dedupeByData {
		buf.WriteString(" AND L.DATA = T.DATA")
	}
	buf.WriteString("\nWHERE T.CHUNK_ID < L.KEEP_CHUNK_ID\n)\n")
	buf.WriteString(final)
	return buf.String()
}

func joinOnColumns(left, right string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, "("+left+"."+col+" = "+right+"."+col+" OR ("+left+"."+col+" IS NULL AND "+right+"."+col+" IS NULL))")
	}
	return strings.Join(parts, " AND ")
}

func qualifyColumns(alias string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, alias+"."+col)
	}
	return strings.Join(parts, ", ")
}

func selectAliasedColumns(alias string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, alias+"."+col+" AS "+col)
	}
	return strings.Join(parts, ", ")
}

func chunkTypeArgs(types []chunk.ChunkType) []any {
	args := make([]any, 0, len(types))
	for _, t := range types {
		args = append(args, t)
	}
	return args
}

func queryxContext(ctx context.Context, db sqlx.QueryerContext, stmt string, args ...any) (*sqlx.Rows, error) {
	if conn, ok := db.(interface{ Rebind(string) string }); ok {
		stmt = conn.Rebind(stmt)
	}
	return db.QueryxContext(ctx, stmt, args...)
}
