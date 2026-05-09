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

package dbase

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

// seedDuplicateMessage inserts two finished sessions, two CMessages chunks and
// two MESSAGE rows that share (ID, payload).  It mimics the state left behind
// by an "archive" run followed by a "resume" run that re-fetched the same
// message because of look-back overlap (rusq/slackdump#633).  The returned
// session IDs are the parent (archive) and child (resume) sessions.
func seedDuplicateMessage(t *testing.T, db *sqlx.DB) (parentID, childID int64) {
	t.Helper()
	ctx := context.Background()
	sr := repository.NewSessionRepository()
	cr := repository.NewChunkRepository()
	mr := repository.NewMessageRepository()

	parentID, err := sr.Insert(ctx, db, &repository.Session{Finished: true, Mode: "archive"})
	require.NoError(t, err)
	childID, err = sr.Insert(ctx, db, &repository.Session{Finished: true, Mode: "resume", ParentID: &parentID})
	require.NoError(t, err)

	channelID := "C123"
	chunkA, err := cr.Insert(ctx, db, &repository.DBChunk{
		SessionID:  parentID,
		TypeID:     chunk.CMessages,
		ChannelID:  &channelID,
		NumRecords: 1,
	})
	require.NoError(t, err)
	chunkB, err := cr.Insert(ctx, db, &repository.DBChunk{
		SessionID:  childID,
		TypeID:     chunk.CMessages,
		ChannelID:  &channelID,
		NumRecords: 1,
	})
	require.NoError(t, err)

	payload := []byte(`{"text":"hello","ts":"1700000000.000100"}`)
	require.NoError(t, mr.Insert(ctx, db, &repository.DBMessage{
		ID: 1700000000000100, ChunkID: chunkA, ChannelID: channelID,
		TS: "1700000000.000100", Index: 0, Text: "hello", Data: payload,
	}))
	require.NoError(t, mr.Insert(ctx, db, &repository.DBMessage{
		ID: 1700000000000100, ChunkID: chunkB, ChannelID: channelID,
		TS: "1700000000.000100", Index: 0, Text: "hello", Data: payload,
	}))
	return parentID, childID
}

func countMessages(t *testing.T, db *sqlx.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.GetContext(context.Background(), &n, "SELECT COUNT(*) FROM MESSAGE"))
	return n
}

// TestDBP_FinishDedupeOnFinish verifies that DBP.Finish runs the deduplication
// pass when configured with WithDedupeOnFinish, and leaves duplicates in place
// otherwise.  This is the wiring test for rusq/slackdump#633; the dedup
// SQL itself is exercised by repository.TestDedupeRepository_Deduplicate.
func TestDBP_FinishDedupeOnFinish(t *testing.T) {
	t.Run("disabled by default keeps duplicates", func(t *testing.T) {
		db := testDB(t)
		seedDuplicateMessage(t, db)
		require.EqualValues(t, 2, countMessages(t, db))

		d, err := New(t.Context(), db, SessionInfo{Mode: "resume"})
		require.NoError(t, err)
		require.NoError(t, d.Finish())

		assert.EqualValues(t, 2, countMessages(t, db),
			"Finish without WithDedupeOnFinish must not touch duplicates")
	})

	t.Run("enabled removes duplicates", func(t *testing.T) {
		db := testDB(t)
		seedDuplicateMessage(t, db)
		require.EqualValues(t, 2, countMessages(t, db))

		d, err := New(t.Context(), db, SessionInfo{Mode: "resume"}, WithDedupeOnFinish(true))
		require.NoError(t, err)
		require.NoError(t, d.Finish())

		assert.EqualValues(t, 1, countMessages(t, db),
			"Finish with WithDedupeOnFinish must collapse identical duplicates")
	})

	t.Run("enabled is a no-op on a clean database", func(t *testing.T) {
		db := testDB(t)
		d, err := New(t.Context(), db, SessionInfo{Mode: "resume"}, WithDedupeOnFinish(true))
		require.NoError(t, err)
		require.NoError(t, d.Finish(), "dedupe on a clean db must not fail Finish")
	})

	t.Run("dedupe failure does not fail Finish", func(t *testing.T) {
		// Regression guard for the contract documented on DBP.Finish: a
		// successful session must never be reported as failed because the
		// opportunistic dedupe pass errored out.  We force a dedupe failure
		// by dropping a table the dedupe SQL relies on after Finalise has
		// already run against SESSION.
		db := testDB(t)
		seedDuplicateMessage(t, db)

		d, err := New(t.Context(), db, SessionInfo{Mode: "resume"}, WithDedupeOnFinish(true))
		require.NoError(t, err)

		// Drop MESSAGE so dedupe's first entity pass blows up; SESSION is
		// untouched so Finalise can still mark the session done.
		_, err = db.ExecContext(t.Context(), "DROP TABLE MESSAGE")
		require.NoError(t, err)

		assert.NoError(t, d.Finish(),
			"dedupe errors must be swallowed: the session is already finalised")
	})

	t.Run("dedupe is skipped on Abort", func(t *testing.T) {
		db := testDB(t)
		seedDuplicateMessage(t, db)
		require.EqualValues(t, 2, countMessages(t, db))

		d, err := New(t.Context(), db, SessionInfo{Mode: "resume"}, WithDedupeOnFinish(true))
		require.NoError(t, err)
		require.NoError(t, d.Abort())

		assert.EqualValues(t, 2, countMessages(t, db),
			"aborted sessions must be left untouched even with WithDedupeOnFinish")
	})
}
