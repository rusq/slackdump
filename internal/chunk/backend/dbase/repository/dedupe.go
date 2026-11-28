package repository

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

type DedupeStats struct {
	Messages       int64
	CanvasMessages int64
	Users          int64
	Channels       int64
	ChannelUsers   int64
	Files          int64
	Chunks         int64
}

func (s DedupeStats) totalRows() int64 {
	return s.Messages + s.CanvasMessages + s.Users + s.Channels + s.ChannelUsers + s.Files
}

type MessageDedupeMode string

const (
	MessageDedupeExact MessageDedupeMode = "exact"
	MessageDedupeKey   MessageDedupeMode = "message-key"
)

func (m *MessageDedupeMode) Set(v string) error {
	mode := MessageDedupeMode(strings.ToLower(v))
	switch mode {
	case MessageDedupeExact, MessageDedupeKey:
		*m = mode
		return nil
	default:
		return fmt.Errorf("unknown message dedupe mode: %s", v)
	}
}

func (m MessageDedupeMode) String() string {
	if m == "" {
		return string(MessageDedupeExact)
	}
	return string(m)
}

type DedupeRepository interface {
	Preview(ctx context.Context, db *sqlx.DB) (DedupeStats, error)
	Deduplicate(ctx context.Context, db *sqlx.DB) (DedupeStats, error)
}

type DedupeOption func(*dedupeRepository)

func WithMessageDedupeMode(mode MessageDedupeMode) DedupeOption {
	return func(r *dedupeRepository) {
		r.messageMode = mode
	}
}

type dedupeRepository struct {
	messageMode MessageDedupeMode
	batchRows   int
}

type dedupeMode int

const (
	dedupeByData dedupeMode = iota
	dedupeByKey
)

//go:generate go tool stringer -trimprefix ent -type entityType
type entityType int8

const (
	entMessages entityType = iota
	entCanvasMessages
	entUsers
	entChannels
	entChannelUsers
	entFiles
)

type dedupeEntity struct {
	etype      entityType
	table      string
	keyColumns []string
	chunkTypes []chunk.ChunkType
	mode       dedupeMode
}

var dedupeEntities = []dedupeEntity{
	{
		etype:      entMessages,
		table:      "MESSAGE",
		keyColumns: []string{"CHANNEL_ID", "ID"},
		chunkTypes: []chunk.ChunkType{chunk.CMessages, chunk.CThreadMessages},
		mode:       dedupeByData,
	},
	{
		etype:      entCanvasMessages,
		table:      "MESSAGE",
		keyColumns: []string{"CHANNEL_ID", "ID"},
		chunkTypes: []chunk.ChunkType{chunk.CCanvasMessages, chunk.CCanvasThreadMessages},
		mode:       dedupeByData,
	},
	{
		etype:      entUsers,
		table:      "S_USER",
		keyColumns: []string{"ID"},
		chunkTypes: []chunk.ChunkType{chunk.CUsers},
		mode:       dedupeByData,
	},
	{
		etype:      entChannels,
		table:      "CHANNEL",
		keyColumns: []string{"ID"},
		chunkTypes: []chunk.ChunkType{chunk.CChannels, chunk.CChannelInfo},
		mode:       dedupeByData,
	},
	{
		etype:      entChannelUsers,
		table:      "CHANNEL_USER",
		keyColumns: []string{"CHANNEL_ID", "USER_ID"},
		chunkTypes: []chunk.ChunkType{chunk.CChannelUsers},
		mode:       dedupeByKey,
	},
	{
		etype:      entFiles,
		table:      "FILE",
		keyColumns: []string{"ID", "CHANNEL_ID", "MESSAGE_ID", "THREAD_ID"},
		chunkTypes: []chunk.ChunkType{chunk.CFiles},
		mode:       dedupeByData,
	},
}

func NewDedupeRepository(opts ...DedupeOption) DedupeRepository {
	r := dedupeRepository{messageMode: MessageDedupeExact}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

func (r dedupeRepository) entities() []dedupeEntity {
	entities := make([]dedupeEntity, len(dedupeEntities))
	copy(entities, dedupeEntities)
	for i := range entities {
		if (entities[i].etype == entMessages || entities[i].etype == entCanvasMessages) && r.messageMode == MessageDedupeKey {
			entities[i].mode = dedupeByKey
		}
	}
	return entities
}

func (r dedupeRepository) Preview(ctx context.Context, db *sqlx.DB) (DedupeStats, error) {
	var counts DedupeStats
	for _, entity := range r.entities() {
		n, err := r.countDuplicates(ctx, db, entity)
		if err != nil {
			return DedupeStats{}, fmt.Errorf("count duplicate %s: %w", entity.etype, err)
		}
		counts.add(entity.etype, n)

		chunks, err := r.prunableChunkIDs(ctx, db, entity)
		if err != nil {
			return DedupeStats{}, fmt.Errorf("count prunable %s chunks: %w", entity.etype, err)
		}
		counts.Chunks += int64(len(chunks))
	}
	return counts, nil
}

func (r dedupeRepository) Deduplicate(ctx context.Context, db *sqlx.DB) (DedupeStats, error) {
	limit := r.batchRows
	if limit == 0 {
		limit = dedupeBatchRows
	}
	if r.messageMode != MessageDedupeKey {
		r.messageMode = MessageDedupeExact
	}

	var owner string
	for {
		progress, err := r.deduplicateBatch(ctx, db, owner, limit)
		if err != nil {
			return DedupeStats{}, err
		}
		if progress.owner == "" {
			return progress.stats, nil
		}
		owner = progress.owner
	}
}

func (r dedupeRepository) deduplicateRange(ctx context.Context, tx *sqlx.Tx, entity dedupeEntity, bounds dedupeRange, stats *DedupeStats) error {
	rowsRemoved, touchedChunks, err := deleteDuplicates(ctx, tx, entity, bounds)
	if err != nil {
		return fmt.Errorf("delete duplicate %s: %w", entity.etype, err)
	}
	stats.add(entity.etype, rowsRemoved)

	chunksRemoved, err := pruneTouchedChunks(ctx, tx, entity, touchedChunks)
	if err != nil {
		return fmt.Errorf("delete prunable %s chunks: %w", entity.etype, err)
	}
	stats.Chunks += chunksRemoved
	return nil
}

func (s *DedupeStats) add(entType entityType, n int64) {
	switch entType {
	case entMessages:
		s.Messages += n
	case entCanvasMessages:
		s.CanvasMessages += n
	case entUsers:
		s.Users += n
	case entChannels:
		s.Channels += n
	case entChannelUsers:
		s.ChannelUsers += n
	case entFiles:
		s.Files += n
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
	rows, err := queryxContext(ctx, db, stmt, chunkTypeArgs(entity.chunkTypes)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan prunable %s chunk: %w", entity.etype, err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func deleteDuplicates(ctx context.Context, tx *sqlx.Tx, entity dedupeEntity, bounds dedupeRange) (int64, []int64, error) {
	stmt := buildDeleteDuplicatesStmt(entity, bounds)
	args := bounds.appendArgs(chunkTypeArgs(entity.chunkTypes))
	rows, err := queryxContext(ctx, tx, stmt, args...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var removed int64
	touched := make(map[int64]struct{})
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, nil, fmt.Errorf("scan removed %s chunk: %w", entity.etype, err)
		}
		touched[id] = struct{}{}
		removed++
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return removed, slices.Sorted(maps.Keys(touched)), nil
}

func pruneTouchedChunks(ctx context.Context, tx *sqlx.Tx, entity dedupeEntity, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// batchSize limits the number of chunk IDs per DELETE statement to stay
	// within SQLite's SQLITE_MAX_VARIABLE_NUMBER limit.
	const batchSize = 10000
	typeArgs := chunkTypeArgs(entity.chunkTypes)
	typePlaceholders := strings.Join(placeholders(entity.chunkTypes), ",")

	var totalAffected int64
	for start := 0; start < len(ids); start += batchSize {
		end := min(start+batchSize, len(ids))
		batch := ids[start:end]

		var buf strings.Builder
		buf.WriteString("DELETE FROM CHUNK WHERE ID IN (")
		buf.WriteString(strings.Join(placeholders(batch), ","))
		buf.WriteString(") AND TYPE_ID IN (")
		buf.WriteString(typePlaceholders)
		// Probe every remaining key in the touched chunk, including keys
		// outside this range. Pre-existing empty chunks are never touched.
		fmt.Fprintf(&buf, ") AND NOT EXISTS (SELECT 1 FROM %[1]s INDEXED BY %[1]s_CHUNK_ID_IDX WHERE CHUNK_ID = CHUNK.ID)", entity.table)

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
	buf.WriteString("SELECT C.ID\nFROM eligible_chunks C\n")
	buf.WriteString("LEFT JOIN ")
	buf.WriteString(entity.table)
	buf.WriteString(" T ON T.CHUNK_ID = C.ID\n")
	buf.WriteString("LEFT JOIN duplicates D ON D.CHUNK_ID = T.CHUNK_ID")
	if len(entity.keyColumns) > 0 {
		buf.WriteString(" AND ")
		buf.WriteString(joinOnColumns("D", "T", entity.keyColumns))
	}
	buf.WriteString("\nGROUP BY C.ID\nHAVING COUNT(T.CHUNK_ID) > 0 AND COUNT(T.CHUNK_ID) = COUNT(D.CHUNK_ID)")
	return buf.String()
}

func (e dedupeEntity) groupColumns(alias string) string {
	keys := qualifyColumns(alias, e.keyColumns)
	if e.mode == dedupeByData {
		keys += ", " + alias + ".DATA"
	}
	return keys
}

func buildDeleteDuplicatesStmt(entity dedupeEntity, bounds dedupeRange) string {
	source := entity.table + " T"
	where := "(SELECT TYPE_ID FROM CHUNK WHERE ID = T.CHUNK_ID) IN (" + strings.Join(placeholders(entity.chunkTypes), ",") + ")"
	if predicate := bounds.predicate("T"); predicate != "" {
		// Force the primary-key prefix search even when ANALYZE makes scanning
		// an identity index and then filtering by ID look cheaper to SQLite.
		source += " INDEXED BY " + dedupeRangeIndex(entity.table)
		where += " AND " + predicate
	}
	// Scalar lookups constrain CHUNK by its rowid on each eligible payload
	// row. SQLite cannot reorder a join or build a Bloom filter over CHUNK.
	// COALESCE puts a missing session in the same tier as unfinished ones.
	return fmt.Sprintf(`WITH ranked AS (
		SELECT T.rowid AS RID, ROW_NUMBER() OVER (
			PARTITION BY %s
			ORDER BY COALESCE((
				SELECT FINISHED = TRUE FROM SESSION
				WHERE ID = (SELECT SESSION_ID FROM CHUNK WHERE ID = T.CHUNK_ID)
			), FALSE) DESC, T.CHUNK_ID DESC
		) AS POS
		FROM %s
		WHERE %s
	)
	DELETE FROM %s WHERE rowid IN (SELECT RID FROM ranked WHERE POS > 1)
	RETURNING CHUNK_ID`, entity.groupColumns("T"), source, where, entity.table)
}

func withDuplicateRows(entity dedupeEntity, final string) string {
	var buf strings.Builder
	buf.WriteString("WITH eligible_chunks AS NOT MATERIALIZED (\n")
	buf.WriteString("SELECT ID, SESSION_ID FROM CHUNK WHERE TYPE_ID IN (")
	buf.WriteString(strings.Join(placeholders(entity.chunkTypes), ","))
	buf.WriteString(")\n), ")
	buf.WriteString("latest AS (\nSELECT ")
	buf.WriteString(selectAliasedColumns("T", entity.keyColumns))
	if entity.mode == dedupeByData {
		buf.WriteString(", T.DATA AS DATA")
	}
	buf.WriteString(", COALESCE(MAX(CASE WHEN S.FINISHED = TRUE THEN T.CHUNK_ID END), MAX(T.CHUNK_ID)) AS KEEP_CHUNK_ID\n")
	buf.WriteString("FROM ")
	buf.WriteString(entity.table)
	buf.WriteString(" T\nJOIN eligible_chunks C ON C.ID = T.CHUNK_ID\n")
	buf.WriteString("LEFT JOIN SESSION S ON S.ID = C.SESSION_ID\nGROUP BY ")
	buf.WriteString(entity.groupColumns("T"))
	buf.WriteString("\n),\nduplicates AS (\nSELECT T.")
	buf.WriteString(strings.Join(entity.keyColumns, ", T."))
	buf.WriteString(", T.CHUNK_ID\n")
	buf.WriteString("FROM ")
	buf.WriteString(entity.table)
	buf.WriteString(" T\nJOIN eligible_chunks C ON C.ID = T.CHUNK_ID\nJOIN latest L ON ")
	buf.WriteString(joinOnColumns("T", "L", entity.keyColumns))
	if entity.mode == dedupeByData {
		buf.WriteString(" AND L.DATA = T.DATA")
	}
	buf.WriteString("\nWHERE T.CHUNK_ID <> L.KEEP_CHUNK_ID\n)\n")
	buf.WriteString(final)
	return buf.String()
}

func joinOnColumns(left, right string, cols []string) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, left+"."+col+" IS "+right+"."+col)
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
