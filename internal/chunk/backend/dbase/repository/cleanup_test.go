package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func TestCleanupRepository_CountUnfinishedSessions(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	insertSessionForTest(t, db, 1, false, nil)
	insertSessionForTest(t, db, 2, true, nil)
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 12, 1, chunk.CUsers)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)

	sessionCount, err := repo.CountUnfinishedSessions(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), sessionCount)
}

func TestCleanupRepository_CountUnfinishedChunks(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	insertSessionForTest(t, db, 1, false, nil)
	insertSessionForTest(t, db, 2, true, nil)
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 12, 1, chunk.CUsers)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)

	chunkCount, err := repo.CountUnfinishedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), chunkCount)
}

func TestCleanupRepository_CleanupUnfinishedSessions(t *testing.T) {
	t.Run("removes unfinished session data", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewCleanupRepository()

		insertSessionForTest(t, db, 1, false, nil)
		insertSessionForTest(t, db, 2, true, nil)
		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 12, 1, chunk.CThreadMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"remove"}`))
		insertMessageWithChunkForTest(t, db, 101, 12, []byte(`{"text":"remove thread"}`))
		insertMessageWithChunkForTest(t, db, 200, 21, []byte(`{"text":"keep"}`))

		result, err := repo.CleanupUnfinishedSessions(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.SessionsRemoved)
		assert.Equal(t, int64(2), result.ChunksRemoved)
		assert.Equal(t, int64(2), result.PayloadRemoved["MESSAGE"])
		assert.Equal(t, int64(2), result.PayloadRemoved.Total())

		verifySessionCountForTest(t, db, 1)
		verifyChunkCountForTest(t, db, 1)
		verifyMessageCountForTest(t, db, 1)
	})

	t.Run("removes payload rows with foreign keys disabled", func(t *testing.T) {
		// Regression test for #755: cleanup must not rely on ON DELETE CASCADE.
		db := testConnNoFK(t)
		ctx := context.Background()
		repo := NewCleanupRepository()

		seedPayloadForTest(t, db)

		result, err := repo.CleanupUnfinishedSessions(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.SessionsRemoved)
		assert.Equal(t, int64(2), result.ChunksRemoved)
		assert.Equal(t, int64(6), result.PayloadRemoved.Total())

		verifySessionCountForTest(t, db, 1)
		verifyChunkCountForTest(t, db, 5)
		verifyMessageCountForTest(t, db, 1)
		verifyTableCountForTest(t, db, "CHANNEL", 1)
		verifyTableCountForTest(t, db, "FILE", 1)
		verifyTableCountForTest(t, db, "S_USER", 1)
		verifyTableCountForTest(t, db, "CHANNEL_USER", 1)

		orphans, err := repo.CountOrphanedPayload(ctx, db)
		require.NoError(t, err)
		assert.Zero(t, orphans.Total(), "cleanup must not leave orphaned payload rows")
	})

	t.Run("removes multiple unfinished sessions", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewCleanupRepository()

		insertSessionForTest(t, db, 1, false, nil)
		insertSessionForTest(t, db, 2, false, new(int64(1)))
		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)

		result, err := repo.CleanupUnfinishedSessions(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, int64(2), result.SessionsRemoved)
		assert.Equal(t, int64(2), result.ChunksRemoved)
		assert.Zero(t, result.PayloadRemoved.Total())

		verifySessionCountForTest(t, db, 0)
		verifyChunkCountForTest(t, db, 0)
	})

	t.Run("returns zero when all sessions are finished", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewCleanupRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertChunkForTest(t, db, 11, 1, chunk.CMessages)

		result, err := repo.CleanupUnfinishedSessions(ctx, db)
		require.NoError(t, err)
		assert.Zero(t, result.SessionsRemoved)
		assert.Zero(t, result.ChunksRemoved)
		assert.Zero(t, result.PayloadRemoved.Total())

		verifySessionCountForTest(t, db, 1)
		verifyChunkCountForTest(t, db, 1)
	})
}

func TestCleanupRepository_CountUnfinishedPayload(t *testing.T) {
	db := testConnNoFK(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	seedPayloadForTest(t, db)

	counts, err := repo.CountUnfinishedPayload(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, TableCounts{
		"MESSAGE":        2,
		"CHANNEL":        1,
		"FILE":           1,
		"WORKSPACE":      0,
		"S_USER":         1,
		"CHANNEL_USER":   1,
		"SEARCH_MESSAGE": 0,
		"SEARCH_FILE":    0,
	}, counts)
}

func TestCleanupRepository_CountOrphanedPayload(t *testing.T) {
	db := testConnNoFK(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	seedPayloadForTest(t, db)
	insertOrphansForTest(t, db)

	counts, err := repo.CountOrphanedPayload(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts["MESSAGE"])
	assert.Equal(t, int64(1), counts["S_USER"])
	assert.Equal(t, int64(3), counts.Total())
}

func TestCleanupRepository_DeleteOrphanedPayload(t *testing.T) {
	db := testConnNoFK(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	seedPayloadForTest(t, db)
	insertOrphansForTest(t, db)

	removed, err := repo.DeleteOrphanedPayload(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), removed["MESSAGE"])
	assert.Equal(t, int64(1), removed["S_USER"])
	assert.Equal(t, int64(3), removed.Total())

	// non-orphaned rows are retained
	verifyMessageCountForTest(t, db, 3)
	verifyTableCountForTest(t, db, "S_USER", 2)

	left, err := repo.CountOrphanedPayload(ctx, db)
	require.NoError(t, err)
	assert.Zero(t, left.Total())
}

func Test_deleteUnfinishedSession_notFound(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()

	tx, err := db.BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	_, err = deleteUnfinishedSession(ctx, tx, 999) // non-existent session ID
	assert.Error(t, err)
	assert.NotErrorIs(t, err, sql.ErrNoRows)
}

func insertSessionForTest(t *testing.T, db *sqlx.DB, id int64, finished bool, parentID *int64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO SESSION (ID, PAR_SESSION_ID, FINISHED, MODE)
		VALUES (?, ?, ?, 'archive')`,
		id, parentID, finished)
	require.NoError(t, err)
}

func insertChunkForTest(t *testing.T, db *sqlx.DB, chunkID, sessionID int64, typeID chunk.ChunkType) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO CHUNK (ID, SESSION_ID, TYPE_ID, UNIX_TS)
		VALUES (?, ?, ?, ?)`,
		chunkID, sessionID, typeID, 1000000000)
	require.NoError(t, err)
}

func insertMessageWithChunkForTest(t *testing.T, db *sqlx.DB, msgID, chunkID int64, data []byte) {
	t.Helper()
	insertMessageInChannelWithChunkForTest(t, db, msgID, chunkID, "C001", data)
}

func insertMessageInChannelWithChunkForTest(t *testing.T, db *sqlx.DB, msgID, chunkID int64, channelID string, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
		VALUES (?, ?, ?, '1000000000.000001', 0, ?)`,
		msgID, chunkID, channelID, data)
	require.NoError(t, err)
}

func verifySessionCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM SESSION").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

func verifyChunkCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM CHUNK").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

func verifyMessageCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM MESSAGE").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

// testConnNoFK returns a single-connection in-memory database with foreign
// keys disabled, so that ON DELETE CASCADE does not fire.  This mirrors
// databases opened without the foreign_keys pragma.
func testConnNoFK(t *testing.T) *sqlx.DB {
	t.Helper()
	db := testConn(t)
	db.SetMaxOpenConns(1) // in-memory: every connection is a separate database
	_, err := db.Exec("PRAGMA foreign_keys = OFF")
	require.NoError(t, err)
	var fk int
	require.NoError(t, db.QueryRowx("PRAGMA foreign_keys").Scan(&fk))
	require.Equal(t, 0, fk)
	return db
}

// seedPayloadForTest creates an unfinished session 1 (chunks 11, 12) and a
// finished session 2 (chunks 21-25), each with payload rows.
func seedPayloadForTest(t *testing.T, db *sqlx.DB) {
	t.Helper()
	insertSessionForTest(t, db, 1, false, nil)
	insertSessionForTest(t, db, 2, true, nil)

	// unfinished session
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 12, 1, chunk.CUsers)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"remove"}`))
	insertMessageWithChunkForTest(t, db, 101, 11, []byte(`{"text":"remove too"}`))
	insertChannelWithChunkForTest(t, db, "C001", 11, "general", []byte(`{}`))
	insertFileWithChunkForTest(t, db, "F001", 11, "C001", nil, nil, []byte(`{}`))
	insertUserWithChunkForTest(t, db, "U001", 12, "remove", []byte(`{}`))
	insertChannelUserWithChunkForTest(t, db, "C001", "U001", 12)

	// finished session
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertChunkForTest(t, db, 22, 2, chunk.CChannelInfo)
	insertChunkForTest(t, db, 23, 2, chunk.CFiles)
	insertChunkForTest(t, db, 24, 2, chunk.CUsers)
	insertChunkForTest(t, db, 25, 2, chunk.CChannelUsers)
	insertMessageWithChunkForTest(t, db, 200, 21, []byte(`{"text":"keep"}`))
	insertChannelWithChunkForTest(t, db, "C001", 22, "general", []byte(`{}`))
	insertFileWithChunkForTest(t, db, "F001", 23, "C001", nil, nil, []byte(`{}`))
	insertUserWithChunkForTest(t, db, "U001", 24, "keep", []byte(`{}`))
	insertChannelUserWithChunkForTest(t, db, "C001", "U001", 25)
}

// insertOrphansForTest inserts payload rows that reference the non-existent
// chunk 999.
func insertOrphansForTest(t *testing.T, db *sqlx.DB) {
	t.Helper()
	insertMessageWithChunkForTest(t, db, 900, 999, []byte(`{"text":"orphan"}`))
	insertMessageWithChunkForTest(t, db, 901, 999, []byte(`{"text":"orphan 2"}`))
	insertUserWithChunkForTest(t, db, "U999", 999, "orphan", []byte(`{}`))
}

func verifyTableCountForTest(t *testing.T, db *sqlx.DB, table string, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count, table)
}
