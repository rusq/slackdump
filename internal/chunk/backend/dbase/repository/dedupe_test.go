package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func TestMessageDedupeRepository_CountDuplicateMessages(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewMessageDedupeRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, ptr(int64(1)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
	insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

	count, err := repo.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMessageDedupeRepository_DeduplicateMessagesKeepsLatest(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewMessageDedupeRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, ptr(int64(1)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
	insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

	result, err := repo.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, MessageDedupeResult{MessagesRemoved: 1, ChunksRemoved: 1}, result)

	verifyChunkCountForTest(t, db, 1)
	verifyMessageCountForTest(t, db, 1)
	verifyMessageChunkForTest(t, db, 100, 21)
}

func TestMessageDedupeRepository_PreservesEditedMessages(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewMessageDedupeRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, ptr(int64(1)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"old"}`))
	insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"edited"}`))

	count, err := repo.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	result, err := repo.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, MessageDedupeResult{}, result)

	verifyChunkCountForTest(t, db, 2)
	verifyMessageCountForTest(t, db, 2)
}

func TestMessageDedupeRepository_PrunesOnlyDuplicateOnlyChunks(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewMessageDedupeRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, ptr(int64(1)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 12, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
	insertMessageWithChunkForTest(t, db, 101, 12, []byte(`{"text":"keep"}`))
	insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

	chunkCount, err := repo.CountPrunableMessageChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), chunkCount)

	result, err := repo.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, MessageDedupeResult{MessagesRemoved: 1, ChunksRemoved: 1}, result)

	verifyChunkCountForTest(t, db, 2)
	verifyMessageCountForTest(t, db, 2)
}

func TestMessageDedupeRepository_DeduplicatesAcrossMultipleSessions(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	repo := NewMessageDedupeRepository()

	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, ptr(int64(1)))
	insertSessionForTest(t, db, 3, true, ptr(int64(2)))
	insertChunkForTest(t, db, 11, 1, chunk.CMessages)
	insertChunkForTest(t, db, 21, 2, chunk.CMessages)
	insertChunkForTest(t, db, 31, 3, chunk.CMessages)
	insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
	insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))
	insertMessageWithChunkForTest(t, db, 100, 31, []byte(`{"text":"same"}`))

	result, err := repo.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, MessageDedupeResult{MessagesRemoved: 2, ChunksRemoved: 2}, result)

	verifyChunkCountForTest(t, db, 1)
	verifyMessageChunkForTest(t, db, 100, 31)
}

func verifyMessageChunkForTest(t *testing.T, db *sqlx.DB, msgID, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM MESSAGE WHERE ID = ?", msgID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}
