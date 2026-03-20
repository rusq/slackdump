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

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountUnreferencedChunks(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 3)
	setupChunkForTest(t, db, 3, 2)

	insertMessageChunkForTest(t, db, 1)

	count, err := vr.CountUnreferencedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestPruneUnreferencedChunks(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 3)
	setupChunkForTest(t, db, 3, 2)

	insertMessageChunkForTest(t, db, 1)

	deleted, err := vr.PruneUnreferencedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	verifyChunkCountForTest(t, db, 1)
}

func TestPruneUnreferencedChunks_afterDedupe(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 1)

	insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{"text":"hello"}`))
	insertMessageWithChunkForTest(t, db, 100, 2, []byte(`{"text":"hello"}`))

	deleted, err := vr.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	pruned, err := vr.PruneUnreferencedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pruned)

	verifyChunkCountForTest(t, db, 1)
}

func TestDeduplicateMessages(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 0)

	insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{"text":"hello"}`))
	insertMessageWithChunkForTest(t, db, 100, 2, []byte(`{"text":"hello"}`))
	insertMessageWithChunkForTest(t, db, 101, 1, []byte(`{"text":"world"}`))
	insertMessageWithChunkForTest(t, db, 101, 2, []byte(`{"text":"edited"}`))

	count, err := vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	deleted, err := vr.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	count, err = vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	verifyMessageCountForTest(t, db, 3)
}

func TestDeduplicateMessages_keepsOldest(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 0)
	setupChunkForTest(t, db, 3, 0)

	insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{"text":"oldest"}`))
	insertMessageWithChunkForTest(t, db, 100, 2, []byte(`{"text":"oldest"}`))
	insertMessageWithChunkForTest(t, db, 100, 3, []byte(`{"text":"oldest"}`))

	count, err := vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	deleted, err := vr.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	count, err = vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	verifyMessageCountForTest(t, db, 1)
}

func TestDeduplicateMessages_differentData(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	setupChunkForTest(t, db, 2, 0)

	insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{"text":"hello"}`))
	insertMessageWithChunkForTest(t, db, 100, 2, []byte(`{"text":"different"}`))

	count, err := vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestDeduplicateUsers(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 3)
	setupChunkForTest(t, db, 2, 3)

	insertUserWithChunkForTest(t, db, "U001", 1, []byte(`{"name":"Alice"}`))
	insertUserWithChunkForTest(t, db, "U001", 2, []byte(`{"name":"Alice"}`))
	insertUserWithChunkForTest(t, db, "U002", 1, []byte(`{"name":"Bob"}`))
	insertUserWithChunkForTest(t, db, "U002", 2, []byte(`{"name":"Bob edited"}`))

	count, err := vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	deleted, err := vr.DeduplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	count, err = vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	verifyUserCountForTest(t, db, 3)
}

func TestDeduplicateUsers_keepsOldest(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 3)
	setupChunkForTest(t, db, 2, 3)
	setupChunkForTest(t, db, 3, 3)

	insertUserWithChunkForTest(t, db, "U001", 1, []byte(`{"name":"Alice"}`))
	insertUserWithChunkForTest(t, db, "U001", 2, []byte(`{"name":"Alice"}`))
	insertUserWithChunkForTest(t, db, "U001", 3, []byte(`{"name":"Alice"}`))

	count, err := vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	deleted, err := vr.DeduplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	count, err = vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	verifyUserCountForTest(t, db, 1)
}

func TestDeduplicateUsers_differentData(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 3)
	setupChunkForTest(t, db, 2, 3)

	insertUserWithChunkForTest(t, db, "U001", 1, []byte(`{"name":"Alice"}`))
	insertUserWithChunkForTest(t, db, "U001", 2, []byte(`{"name":"Bob"}`))

	count, err := vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestCountUnreferencedChunks_empty(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	insertMessageChunkForTest(t, db, 1)

	count, err := vr.CountUnreferencedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestPruneUnreferencedChunks_noUnreferenced(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)
	insertMessageChunkForTest(t, db, 1)

	deleted, err := vr.PruneUnreferencedChunks(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	verifyChunkCountForTest(t, db, 1)
}

func TestDeduplicateMessages_empty(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 0)

	insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{"text":"hello"}`))

	count, err := vr.CountDuplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	deleted, err := vr.DeduplicateMessages(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func TestDeduplicateUsers_empty(t *testing.T) {
	db := testConn(t)
	ctx := context.Background()
	vr := NewVacuumRepository()

	setupSession(t, db)
	setupChunkForTest(t, db, 1, 3)

	insertUserWithChunkForTest(t, db, "U001", 1, []byte(`{"name":"Alice"}`))

	count, err := vr.CountDuplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	deleted, err := vr.DeduplicateUsers(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}

func setupSession(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive')`)
	require.NoError(t, err)
}

func setupChunkForTest(t *testing.T, db *sqlx.DB, chunkID int64, typeID int) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO CHUNK (ID, SESSION_ID, TYPE_ID, UNIX_TS) VALUES (?, 1, ?, ?)`,
		chunkID, typeID, 1000000000)
	require.NoError(t, err)
}

func insertMessageChunkForTest(t *testing.T, db *sqlx.DB, chunkID int64) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
		VALUES (?, ?, 'C001', '1000000000.000001', 0, '[]')`,
		chunkID*1000000000000, chunkID)
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

func insertUserWithChunkForTest(t *testing.T, db *sqlx.DB, userID string, chunkID int64, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO S_USER (ID, CHUNK_ID, USERNAME, IDX, DATA)
		VALUES (?, ?, ?, 0, ?)`,
		userID, chunkID, userID, data)
	require.NoError(t, err)
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

func verifyUserCountForTest(t *testing.T, db *sqlx.DB, expected int) {
	t.Helper()
	var count int64
	err := db.QueryRowxContext(context.Background(), "SELECT COUNT(*) FROM S_USER").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(expected), count)
}
