package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func TestCleanupRepository_Counts(t *testing.T) {
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

	chunkCount, err := repo.CountUnfinishedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), chunkCount)
}

func TestCleanupRepository_CleanupUnfinishedSessions(t *testing.T) {
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
	assert.Equal(t, CleanupResult{SessionsRemoved: 1, ChunksRemoved: 2}, result)

	verifySessionCountForTest(t, db, 1)
	verifyChunkCountForTest(t, db, 1)
	verifyMessageCountForTest(t, db, 1)
}

func TestCleanupRepository_CleanupMultipleSessions(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	insertSessionForTest(t, db, 1, false, nil)
	insertSessionForTest(t, db, 2, false, ptr(int64(1)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)

	result, err := repo.CleanupUnfinishedSessions(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, CleanupResult{SessionsRemoved: 2, ChunksRemoved: 2}, result)

	verifySessionCountForTest(t, db, 0)
	verifyChunkCountForTest(t, db, 0)
}

func TestCleanupRepository_NoUnfinishedSessions(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewCleanupRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)

	result, err := repo.CleanupUnfinishedSessions(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, CleanupResult{}, result)

	verifySessionCountForTest(t, db, 1)
	verifyChunkCountForTest(t, db, 1)
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
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
		VALUES (?, ?, 'C001', '1000000000.000001', 0, ?)`,
		msgID, chunkID, data)
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
