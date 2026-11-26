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
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
	t.Run("Migrate", func(t *testing.T) {
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		if err := Migrate(t.Context(), db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}
	})

	for _, upgrade := range []bool{false, true} {
		t.Run(fmt.Sprintf("message dedupe index upgrade=%t", upgrade), func(t *testing.T) {
			db, err := sql.Open(Driver, ":memory:")
			require.NoError(t, err)
			defer db.Close()
			ctx := t.Context()
			if upgrade {
				require.NoError(t, goose.UpToContext(ctx, db, "migrations", 20260725000000))
				_, err = db.ExecContext(ctx, `
					INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive');
					INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (1, 0, 1, 0);
					INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
					VALUES (100, 1, 'C1', '0.000100', 0, X'7B7D');`)
				require.NoError(t, err)
			}
			require.NoError(t, Migrate(ctx, db, false))
			var columns []string
			require.NoError(t, sqlx.NewDb(db, Driver).SelectContext(ctx, &columns,
				`SELECT name FROM pragma_index_info('MESSAGE_DEDUPE_IDX') ORDER BY seqno`))
			require.Equal(t, []string{"CHANNEL_ID", "ID", "CHUNK_ID"}, columns)
			if upgrade {
				var data []byte
				require.NoError(t, db.QueryRowContext(ctx,
					`SELECT DATA FROM MESSAGE WHERE ID=100 AND CHUNK_ID=1 AND CHANNEL_ID='C1'`).Scan(&data))
				require.Equal(t, []byte(`{}`), data)
			}
			// Down removes only the new index; existing indexes and rows survive.
			require.NoError(t, goose.DownToContext(ctx, db, "migrations", 20260725000000))
			var count int
			require.NoError(t, db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM sqlite_master WHERE name='MESSAGE_DEDUPE_IDX'`).Scan(&count))
			require.Zero(t, count)
			require.NoError(t, db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM sqlite_master WHERE name IN ('MESSAGE_I1', 'MESSAGE_CHUNK_ID_IDX')`).Scan(&count))
			require.Equal(t, 2, count)
			require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM MESSAGE`).Scan(&count))
			if upgrade {
				require.Equal(t, 1, count)
			} else {
				require.Zero(t, count)
			}
			require.NoError(t, Migrate(ctx, db, false))
		})
	}

	t.Run("dedupe checkpoint upgrade and rollback", func(t *testing.T) {
		db, err := sqlx.Open(Driver, ":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		ctx := t.Context()
		const oldVersion = 20261019000000
		require.NoError(t, goose.UpToContext(ctx, db.DB, "migrations", oldVersion))
		insertSessionForTest(t, db, 1, false, nil)
		_, err = db.Exec(`INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (1, 0, 1, 0)`)
		require.NoError(t, err)
		insertMessageWithChunkForTest(t, db, 100, 1, []byte(`{}`))

		require.NoError(t, Migrate(ctx, db.DB, false))
		seedCheckpoint := func() {
			t.Helper()
			tx, err := db.Beginx()
			require.NoError(t, err)
			defer tx.Rollback()
			p := dedupeCheckpoint{mode: MessageDedupeExact.String(), owner: "test", key: dedupeKey{id: int64(200)}, countersScope: "full_pass"}
			require.NoError(t, p.save(ctx, tx))
			require.NoError(t, tx.Commit())
		}
		var count int
		for _, mutation := range []string{
			"UPDATE SESSION SET FINISHED = TRUE WHERE ID = 1",
			"INSERT INTO SESSION (ID, MODE) VALUES (2, 'resume')",
			"UPDATE CHUNK SET SESSION_ID = 2 WHERE ID = 1",
			"DELETE FROM SESSION WHERE ID = 1",
			"INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (2, 0, 2, 0)",
			"DELETE FROM CHUNK WHERE ID = 2",
		} {
			seedCheckpoint()
			_, err := db.Exec("UPDATE SESSION SET UPDATED_AT = CURRENT_TIMESTAMP")
			require.NoError(t, err)
			require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM DEDUPE_CHECKPOINT"))
			require.Equal(t, 1, count)
			_, err = db.Exec(mutation)
			require.NoError(t, err)
			require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM DEDUPE_CHECKPOINT"))
			require.Zero(t, count, mutation)
		}

		seedCheckpoint()
		require.NoError(t, goose.DownToContext(ctx, db.DB, "migrations", oldVersion))
		require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM sqlite_schema WHERE name LIKE 'DEDUPE_%'"))
		require.Zero(t, count)
		verifyMessageChunkForTest(t, db, 100, 1)
		require.NoError(t, Migrate(ctx, db.DB, false))
		require.NoError(t, db.Get(&count, "SELECT COUNT(*) FROM DEDUPE_CHECKPOINT"))
		require.Zero(t, count)
	})

	t.Run("dedupe counter upgrade preserves legacy checkpoint", func(t *testing.T) {
		db, err := sqlx.Open(Driver, ":memory:")
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		ctx := t.Context()
		const oldVersion = 20261126000000
		require.NoError(t, goose.UpToContext(ctx, db.DB, "migrations", oldVersion))
		populateCheckpointDedupeTest(t, db)
		_, err = db.Exec(`INSERT INTO DEDUPE_CHECKPOINT
			(ID, MODE, OWNER, ENTITY, KEY1, KEY2, MESSAGES, CANVAS_MESSAGES, USERS, CHANNELS, CHANNEL_USERS, FILES, CHUNKS)
			VALUES (1, 'exact', 'legacy', 0, 100, '', 5, 0, 0, 0, 0, 0, 2)`)
		require.NoError(t, err)

		readCheckpoint := func() dedupeCheckpoint {
			t.Helper()
			tx, err := db.BeginTxx(ctx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			got, err := readDedupeCheckpoint(ctx, tx)
			require.NoError(t, err)
			return got
		}

		require.NoError(t, Migrate(ctx, db.DB, false))
		migrated := readCheckpoint()
		require.Equal(t, "legacy", migrated.owner)
		require.Equal(t, int64(100), migrated.key.id)
		require.Equal(t, DedupeStats{Messages: 5, Chunks: 2}, migrated.stats)
		require.Equal(t, "since_upgrade", migrated.countersScope)
		require.Zero(t, migrated.rowsExamined)
		require.Zero(t, migrated.batchesCommitted)

		repo := dedupeRepository{messageMode: MessageDedupeExact}
		_, err = repo.deduplicateBatch(ctx, db, "", 2)
		require.NoError(t, err)
		continued := readCheckpoint()
		require.Equal(t, "since_upgrade", continued.countersScope)
		require.EqualValues(t, 6, continued.rowsExamined)
		require.EqualValues(t, 1, continued.batchesCommitted)

		require.NoError(t, goose.DownToContext(ctx, db.DB, "migrations", oldVersion))
		var count int
		require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM pragma_table_info('DEDUPE_CHECKPOINT')
			WHERE name IN ('ROWS_EXAMINED', 'BATCHES_COMMITTED', 'COUNTERS_SCOPE')`))
		require.Zero(t, count)
		require.NoError(t, Migrate(ctx, db.DB, false))
		restored := readCheckpoint()
		require.Equal(t, continued.owner, restored.owner)
		require.Equal(t, continued.key, restored.key)
		require.Equal(t, continued.stats, restored.stats)
		require.Equal(t, "since_upgrade", restored.countersScope)
		require.Zero(t, restored.rowsExamined)
		require.Zero(t, restored.batchesCommitted)
	})

	t.Run("adds canvas message types", func(t *testing.T) {
		db, err := sql.Open(Driver, ":memory:")
		require.NoError(t, err)
		defer db.Close()
		require.NoError(t, Migrate(t.Context(), db, true))

		rows, err := db.QueryContext(t.Context(), `SELECT ID, NAME FROM TYPES WHERE ID IN (12, 13) ORDER BY ID`)
		require.NoError(t, err)
		defer rows.Close()
		var got []string
		for rows.Next() {
			var id int
			var name string
			require.NoError(t, rows.Scan(&id, &name))
			got = append(got, name)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"CANVAS_MESSAGES", "CANVAS_THREAD_MESSAGES"}, got)
	})

	t.Run("backfills file size from json", func(t *testing.T) {
		ctx := context.Background()
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		const beforeFileSizeMigration = int64(20260307000000)
		if err := goose.UpToContext(ctx, db, "migrations", beforeFileSizeMigration); err != nil {
			t.Fatalf("goose.UpToContext() err = %v; want nil", err)
		}

		if _, err := db.ExecContext(ctx, `INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive')`); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID, NUM_REC) VALUES (1, 0, 1, 2, 1)`); err != nil {
			t.Fatalf("insert chunk: %v", err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO FILE (ID, CHUNK_ID, CHANNEL_ID, IDX, MODE, FILENAME, URL, DATA)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "F123", 1, "C123", 0, "hosted", "example.txt", "https://example.invalid/example.txt", []byte(`{"id":"F123","size":12345}`)); err != nil {
			t.Fatalf("insert file: %v", err)
		}

		if err := Migrate(ctx, db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}

		var got int64
		if err := db.QueryRowContext(ctx, `SELECT SIZE FROM FILE WHERE ID = ? AND CHUNK_ID = ?`, "F123", 1).Scan(&got); err != nil {
			t.Fatalf("select size: %v", err)
		}
		if got != 12345 {
			t.Fatalf("SIZE = %d; want 12345", got)
		}

		fr := NewFileRepository()
		qx := sqlx.NewDb(db, Driver)
		existing, err := fr.GetByIDAndSize(ctx, qx, "F123", 12345)
		require.NoError(t, err)
		if existing == nil {
			t.Fatal("GetByIDAndSize() = nil; want migrated file")
		}

		missing, err := fr.GetByIDAndSize(ctx, qx, "F123", 12346)
		require.NoError(t, err)
		if missing != nil {
			t.Fatalf("GetByIDAndSize() = %#v; want nil for different size", missing)
		}
	})

	t.Run("empty threads view returns chunk session id", func(t *testing.T) {
		ctx := context.Background()
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		if err := Migrate(ctx, db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}

		if _, err := db.ExecContext(ctx, `INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive')`); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID, NUM_REC) VALUES (1, 0, 1, 0, 1)`); err != nil {
			t.Fatalf("insert chunk: %v", err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, PARENT_ID, THREAD_TS, LATEST_REPLY, IS_PARENT, IDX, DATA)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, int64(1700000000000001), 1, "C123", "1700000000.000001", int64(1700000000000001), "1700000000.000001", "0000000000.000000", true, 0, []byte(`{"type":"message","ts":"1700000000.000001","thread_ts":"1700000000.000001"}`)); err != nil {
			t.Fatalf("insert message: %v", err)
		}

		var got struct {
			SessionID int64  `db:"SESSION_ID"`
			ChunkID   int64  `db:"CHUNK_ID"`
			ChannelID string `db:"CHANNEL_ID"`
			ThreadTS  string `db:"THREAD_TS"`
		}
		require.NoError(t, sqlx.NewDb(db, Driver).GetContext(ctx, &got, `SELECT SESSION_ID, CHUNK_ID, CHANNEL_ID, THREAD_TS FROM V_EMPTY_THREADS`))
		require.EqualValues(t, 1, got.SessionID)
		require.EqualValues(t, 1, got.ChunkID)
		require.Equal(t, "C123", got.ChannelID)
		require.Equal(t, "1700000000.000001", got.ThreadTS)
	})

	t.Run("down after latest migration succeeds with empty threads view", func(t *testing.T) {
		ctx := context.Background()
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		if err := Migrate(ctx, db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("goose.DownContext() err = %v; want nil", err)
		}
	})

	t.Run("down removes size column", func(t *testing.T) {
		ctx := context.Background()
		db, err := sql.Open(Driver, ":memory:")
		if err != nil {
			t.Fatalf("sql.Open() err = %v; want nil", err)
		}
		defer db.Close()

		if err := goose.UpToContext(ctx, db, "migrations", 20260725000000); err != nil {
			t.Fatalf("goose.UpToContext() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("first goose.DownContext() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("goose.DownContext() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("second goose.DownContext() err = %v; want nil", err)
		}

		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('FILE') WHERE name = 'SIZE'`).Scan(&count); err != nil {
			t.Fatalf("pragma_table_info query: %v", err)
		}
		if count != 0 {
			t.Fatalf("SIZE column still exists; count = %d, want 0", count)
		}
	})
}
