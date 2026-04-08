package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func TestDedupeRepository_Preview(t *testing.T) {
	t.Run("counts duplicates across supported entities", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

		insertChunkForTest(t, db, 12, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 12, "same", []byte(`{"name":"same"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CChannels)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelInfo)
		insertChannelWithChunkForTest(t, db, "C100", 13, "same", []byte(`{"name":"same"}`))
		insertChannelWithChunkForTest(t, db, "C100", 23, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 14, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 24, 2, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 14)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 24)

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, []byte(`{"name":"same"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeCounts{
			Messages:     1,
			Users:        1,
			Channels:     1,
			ChannelUsers: 1,
			Files:        1,
			Chunks:       5,
		}, counts)
	})

	t.Run("ignores changed payloads but still deduplicates channel users by key", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"old"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"edited"}`))

		insertChunkForTest(t, db, 12, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 12, "old", []byte(`{"name":"old"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "new", []byte(`{"name":"new"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CChannels)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelInfo)
		insertChannelWithChunkForTest(t, db, "C100", 13, "old", []byte(`{"name":"old"}`))
		insertChannelWithChunkForTest(t, db, "C100", 23, "new", []byte(`{"name":"new"}`))

		insertChunkForTest(t, db, 14, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 24, 2, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 14)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 24)

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, []byte(`{"name":"old"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, []byte(`{"name":"new"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeCounts{
			ChannelUsers: 1,
			Chunks:       1,
		}, counts)
	})
}

func TestDedupeRepository_Deduplicate(t *testing.T) {
	t.Run("deduplicates all supported entities and prunes duplicate-only chunks", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

		insertChunkForTest(t, db, 12, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 12, "same", []byte(`{"name":"same"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CChannels)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelInfo)
		insertChannelWithChunkForTest(t, db, "C100", 13, "same", []byte(`{"name":"same"}`))
		insertChannelWithChunkForTest(t, db, "C100", 23, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 14, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 24, 2, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 14)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 24)

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, []byte(`{"name":"same"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeResult{
			MessagesRemoved:     1,
			UsersRemoved:        1,
			ChannelsRemoved:     1,
			ChannelUsersRemoved: 1,
			FilesRemoved:        1,
			ChunksRemoved:       5,
		}, result)

		verifyChunkCountForTest(t, db, 5)
		verifyMessageCountForTest(t, db, 1)
		verifyUserCountForTest(t, db, 1)
		verifyChannelCountForTest(t, db, 1)
		verifyChannelUserCountForTest(t, db, 1)
		verifyFileCountForTest(t, db, 1)
		verifyMessageChunkForTest(t, db, 100, 21)
		verifyUserChunkForTest(t, db, "U100", 22)
		verifyChannelChunkForTest(t, db, "C100", 23)
		verifyChannelUserChunkForTest(t, db, "C100", "U100", 24)
		verifyFileChunkForTest(t, db, "F100", 25)
	})

	t.Run("preserves changed payloads", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"old"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"edited"}`))

		insertChunkForTest(t, db, 12, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 12, "old", []byte(`{"name":"old"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "new", []byte(`{"name":"new"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CChannels)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelInfo)
		insertChannelWithChunkForTest(t, db, "C100", 13, "old", []byte(`{"name":"old"}`))
		insertChannelWithChunkForTest(t, db, "C100", 23, "new", []byte(`{"name":"new"}`))

		insertChunkForTest(t, db, 14, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 24, 2, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 14)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 24)

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, []byte(`{"name":"old"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, []byte(`{"name":"new"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeResult{
			ChannelUsersRemoved: 1,
			ChunksRemoved:       1,
		}, result)

		verifyChunkCountForTest(t, db, 9)
		verifyMessageCountForTest(t, db, 2)
		verifyUserCountForTest(t, db, 2)
		verifyChannelCountForTest(t, db, 2)
		verifyChannelUserCountForTest(t, db, 1)
		verifyFileCountForTest(t, db, 2)
	})

	t.Run("keeps chunks that still contain non-duplicate rows", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 12, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 101, 12, []byte(`{"text":"keep"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CUsers)
		insertChunkForTest(t, db, 14, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 13, "same", []byte(`{"name":"same"}`))
		insertUserWithChunkForTest(t, db, "U101", 14, "keep", []byte(`{"name":"keep"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 15, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 16, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 15)
		insertChannelUserWithChunkForTest(t, db, "C100", "U101", 16)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 23)

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeResult{
			MessagesRemoved:     1,
			UsersRemoved:        1,
			ChannelUsersRemoved: 1,
			ChunksRemoved:       3,
		}, result)

		verifyChunkCountForTest(t, db, 6)
		verifyMessageCountForTest(t, db, 2)
		verifyUserCountForTest(t, db, 2)
		verifyChannelUserCountForTest(t, db, 2)
	})

	t.Run("deduplicates across multiple sessions", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, ptr(int64(1)))
		insertSessionForTest(t, db, 3, true, ptr(int64(2)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertChunkForTest(t, db, 31, 3, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 31, []byte(`{"text":"same"}`))

		insertChunkForTest(t, db, 12, 1, chunk.CUsers)
		insertChunkForTest(t, db, 22, 2, chunk.CUsers)
		insertChunkForTest(t, db, 32, 3, chunk.CUsers)
		insertUserWithChunkForTest(t, db, "U100", 12, "same", []byte(`{"name":"same"}`))
		insertUserWithChunkForTest(t, db, "U100", 22, "same", []byte(`{"name":"same"}`))
		insertUserWithChunkForTest(t, db, "U100", 32, "same", []byte(`{"name":"same"}`))

		insertChunkForTest(t, db, 13, 1, chunk.CChannelUsers)
		insertChunkForTest(t, db, 23, 2, chunk.CChannelUsers)
		insertChunkForTest(t, db, 33, 3, chunk.CChannelUsers)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 13)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 23)
		insertChannelUserWithChunkForTest(t, db, "C100", "U100", 33)

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeResult{
			MessagesRemoved:     2,
			UsersRemoved:        2,
			ChannelUsersRemoved: 2,
			ChunksRemoved:       6,
		}, result)

		verifyChunkCountForTest(t, db, 3)
		verifyMessageChunkForTest(t, db, 100, 31)
		verifyUserChunkForTest(t, db, "U100", 32)
		verifyChannelUserChunkForTest(t, db, "C100", "U100", 33)
	})
}

func insertUserWithChunkForTest(t *testing.T, db *sqlx.DB, userID string, chunkID int64, username string, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO S_USER (ID, CHUNK_ID, USERNAME, IDX, DATA)
		VALUES (?, ?, ?, 0, ?)`,
		userID, chunkID, username, data)
	require.NoError(t, err)
}

func insertChannelWithChunkForTest(t *testing.T, db *sqlx.DB, channelID string, chunkID int64, name string, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO CHANNEL (ID, CHUNK_ID, NAME, IDX, DATA)
		VALUES (?, ?, ?, 0, ?)`,
		channelID, chunkID, name, data)
	require.NoError(t, err)
}

func insertChannelUserWithChunkForTest(t *testing.T, db *sqlx.DB, channelID, userID string, chunkID int64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO CHANNEL_USER (CHANNEL_ID, USER_ID, CHUNK_ID, IDX)
		VALUES (?, ?, ?, 0)`,
		channelID, userID, chunkID)
	require.NoError(t, err)
}

func insertFileWithChunkForTest(t *testing.T, db *sqlx.DB, fileID string, chunkID int64, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO FILE (ID, CHUNK_ID, CHANNEL_ID, IDX, MODE, FILENAME, URL, DATA, SIZE)
		VALUES (?, ?, 'C001', 0, 'hosted', 'file.txt', 'https://example.com/file', ?, 1)`,
		fileID, chunkID, data)
	require.NoError(t, err)
}

func verifyMessageChunkForTest(t *testing.T, db *sqlx.DB, msgID, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM MESSAGE WHERE ID = ?", msgID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyUserChunkForTest(t *testing.T, db *sqlx.DB, userID string, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM S_USER WHERE ID = ?", userID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyChannelChunkForTest(t *testing.T, db *sqlx.DB, channelID string, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM CHANNEL WHERE ID = ?", channelID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyChannelUserChunkForTest(t *testing.T, db *sqlx.DB, channelID, userID string, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM CHANNEL_USER WHERE CHANNEL_ID = ? AND USER_ID = ?", channelID, userID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyFileChunkForTest(t *testing.T, db *sqlx.DB, fileID string, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM FILE WHERE ID = ?", fileID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyUserCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM S_USER").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

func verifyChannelCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM CHANNEL").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

func verifyChannelUserCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM CHANNEL_USER").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}

func verifyFileCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM FILE").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}
