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

		if err := Migrate(ctx, db, true); err != nil {
			t.Fatalf("Migrate() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("first goose.DownContext() err = %v; want nil", err)
		}

		if err := goose.DownContext(ctx, db, "migrations"); err != nil {
			t.Fatalf("goose.DownContext() err = %v; want nil", err)
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
