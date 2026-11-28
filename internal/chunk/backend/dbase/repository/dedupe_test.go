package repository

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rusq/slackdump/v4/internal/chunk"
)

func TestMessageDedupeMode_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MessageDedupeMode
		wantErr bool
	}{
		{name: "exact", input: "exact", want: MessageDedupeExact},
		{name: "message key", input: "message-key", want: MessageDedupeKey},
		{name: "case insensitive", input: "MESSAGE-KEY", want: MessageDedupeKey},
		{name: "unknown", input: "loose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got MessageDedupeMode
			err := got.Set(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, string(tt.want), fmt.Sprint(got))
		})
	}
}

func TestDedupeRepository_Preview(t *testing.T) {
	for _, mode := range []MessageDedupeMode{MessageDedupeExact, MessageDedupeKey} {
		t.Run("skewed/"+string(mode), func(t *testing.T) {
			db := skewedDedupeFixture(t, 2000)
			before := expectedDedupeMessages(t, db, MessageDedupeExact)
			got, err := NewDedupeRepository(WithMessageDedupeMode(mode)).Preview(t.Context(), db)
			require.NoError(t, err)
			messages := int64(125)
			if mode == MessageDedupeKey {
				messages++
			}
			require.Equal(t, DedupeStats{Messages: messages, ChannelUsers: 1500, Chunks: 4}, got)
			require.Equal(t, before, expectedDedupeMessages(t, db, MessageDedupeExact))
		})
	}
	t.Run("cancelled", func(t *testing.T) {
		db := testConn(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := NewDedupeRepository().Preview(ctx, db)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("counts duplicates across supported entities", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

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
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", nil, nil, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C001", nil, nil, []byte(`{"name":"same"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
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
		insertSessionForTest(t, db, 2, true, new(int64(1)))

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
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", nil, nil, []byte(`{"name":"old"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C001", nil, nil, []byte(`{"name":"new"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
			ChannelUsers: 1,
			Chunks:       1,
		}, counts)
	})

	t.Run("message key mode counts messages with volatile block ids", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository(WithMessageDedupeMode(MessageDedupeKey))

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"dRF"}]}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"0Xuo9"}]}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{Messages: 1, Chunks: 1}, counts)
	})

	t.Run("keeps ordinary and canvas message domains separate", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))
		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 12, 1, chunk.CCanvasMessages)
		insertChunkForTest(t, db, 22, 2, chunk.CCanvasThreadMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 12, []byte(`{"text":"same"}`))
		insertMessageWithChunkForTest(t, db, 100, 22, []byte(`{"text":"same"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{CanvasMessages: 1, Chunks: 1}, counts)

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, counts, result)
		var survivors []int64
		require.NoError(t, db.Select(&survivors, "SELECT CHUNK_ID FROM MESSAGE ORDER BY CHUNK_ID"))
		assert.Equal(t, []int64{11, 22}, survivors)
	})

	t.Run("does not count reshared files in different attachment contexts as duplicates", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", new(int64(100)), nil, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C002", new(int64(200)), nil, []byte(`{"name":"same"}`))

		counts, err := repo.Preview(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{}, counts)
	})
}

func TestDedupeRepository_Deduplicate(t *testing.T) {
	t.Run("logs only successful commits", func(t *testing.T) {
		ctx := t.Context()
		small := testConn(t)
		populateCheckpointDedupeTest(t, small)
		path := filepath.Join(t.TempDir(), "slackdump.sqlite")
		db := testConnDSN(t, DSN(path, ModeDefault))
		populateCheckpointDedupeTest(t, db)
		_, err := db.Exec(`CREATE TRIGGER fail_batch BEFORE DELETE ON MESSAGE WHEN OLD.ID = 200
			BEGIN SELECT RAISE(ABORT, 'interrupt next batch'); END`)
		require.NoError(t, err)

		var output bytes.Buffer
		logger := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
		t.Cleanup(func() { slog.SetDefault(logger) })
		const event = `"msg":"dedupe batch committed"`

		_, err = NewDedupeRepository().Deduplicate(ctx, small)
		require.NoError(t, err)
		first := output.String()
		require.Equal(t, 1, strings.Count(first, event))
		require.Contains(t, first, `"level":"INFO"`)
		require.Contains(t, first, `"tables":["MESSAGE","S_USER","CHANNEL","CHANNEL_USER","FILE"]`)
		require.Contains(t, first, `"batch_rows_examined":34`)
		require.Contains(t, first, `"batch_rows_removed":17`)
		require.Contains(t, first, `"batch_chunks_removed":7`)
		require.Regexp(t, `"batch_elapsed":[1-9][0-9]*`, first)
		require.Contains(t, first, `"rows_examined_total":34`)
		require.Contains(t, first, `"batches_committed_total":1`)
		require.Contains(t, first, `"complete":true`)
		require.NotContains(t, first, `"next_key"`)

		output.Reset()
		batched := dedupeRepository{batchRows: 2}
		_, err = batched.Deduplicate(ctx, db)
		require.ErrorContains(t, err, "interrupt next batch")
		partial := output.String()
		require.Equal(t, 1, strings.Count(partial, event))
		require.Contains(t, partial, `"mode":"exact"`)
		require.Contains(t, partial, `"tables":["MESSAGE"]`)
		require.Contains(t, partial, `"batch_rows_examined":6`)
		require.Contains(t, partial, `"batch_rows_removed":4`)
		require.Contains(t, partial, `"batch_chunks_removed":2`)
		require.Contains(t, partial, `"rows_examined_total":6`)
		require.Contains(t, partial, `"batches_committed_total":1`)
		require.Contains(t, partial, `"removed_rows_total":4`)
		require.Contains(t, partial, `"removed_chunks_total":2`)
		require.Contains(t, partial, `"complete":false`)
		require.Contains(t, partial, `"next_table":"MESSAGE","next_key":[200]`)

		require.NoError(t, db.Close())
		db = testConnDSN(t, DSN(path, ModeRW))
		output.Reset()
		_, err = db.Exec("DROP TRIGGER fail_batch")
		require.NoError(t, err)
		_, err = batched.Deduplicate(ctx, db)
		require.NoError(t, err)
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		require.Greater(t, len(lines), 1)
		final := lines[len(lines)-1]
		require.Contains(t, final, event)
		require.Contains(t, final, `"rows_examined_total":34`)
		require.Contains(t, final, fmt.Sprintf(`"batches_committed_total":%d`, len(lines)+1))
		require.Contains(t, final, `"removed_rows_total":17`)
		require.Contains(t, final, `"removed_chunks_total":7`)
		require.Contains(t, final, `"complete":true`)
		require.NotContains(t, final, `"next_key"`)
	})

	t.Run("ranks missing and unfinished sessions together", func(t *testing.T) {
		db := testConnNoFK(t)
		insertSessionForTest(t, db, 1, false, nil)
		insertSessionForTest(t, db, 2, true, nil)
		payload := []byte(`{}`)
		for _, row := range []struct {
			sessionID, chunkID, messageID int64
		}{
			{1, 11, 100},
			{3, 21, 100},
			{2, 12, 200},
			{3, 22, 200},
		} {
			insertChunkForTest(t, db, row.chunkID, row.sessionID, chunk.CMessages)
			insertMessageWithChunkForTest(t, db, row.messageID, row.chunkID, payload)
		}
		// An already empty chunk must not be pruned by this dedupe pass.
		insertChunkForTest(t, db, 31, 2, chunk.CMessages)

		repo := dedupeRepository{batchRows: 2}
		want := DedupeStats{Messages: 2, Chunks: 2}
		preview, err := repo.Preview(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, want, preview)
		stats, err := repo.Deduplicate(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, want, stats)
		verifyMessageChunkForTest(t, db, 100, 21)
		verifyMessageChunkForTest(t, db, 200, 12)
		var chunks []int64
		require.NoError(t, db.Select(&chunks, "SELECT ID FROM CHUNK ORDER BY ID"))
		require.Equal(t, []int64{12, 21, 31}, chunks)
	})

	t.Run("small archive does not write a checkpoint", func(t *testing.T) {
		db := skewedDedupeFixture(t, 2000)
		_, err := db.Exec(`CREATE TRIGGER reject_checkpoint BEFORE INSERT ON DEDUPE_CHECKPOINT
			BEGIN SELECT RAISE(ABORT, 'unexpected checkpoint'); END`)
		require.NoError(t, err)
		got, err := NewDedupeRepository().Deduplicate(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, DedupeStats{Messages: 125, ChannelUsers: 1500, Chunks: 4}, got)
	})

	for _, mode := range []MessageDedupeMode{MessageDedupeExact, MessageDedupeKey} {
		t.Run("continues committed ranges after reopening/"+string(mode), func(t *testing.T) {
			ctx := t.Context()
			path := filepath.Join(t.TempDir(), "slackdump.sqlite")
			db := testConnDSN(t, DSN(path, ModeDefault))
			var journalMode string
			require.NoError(t, db.Get(&journalMode, "PRAGMA journal_mode = WAL"))
			require.Equal(t, "wal", journalMode)
			populateCheckpointDedupeTest(t, db)

			whole := testConn(t)
			populateCheckpointDedupeTest(t, whole)
			wholeRepo := dedupeRepository{messageMode: mode}
			want, err := wholeRepo.Preview(ctx, whole)
			require.NoError(t, err)
			wholeStats, err := wholeRepo.Deduplicate(ctx, whole)
			require.NoError(t, err)
			require.Equal(t, want, wholeStats)

			_, err = db.Exec(`CREATE TRIGGER fail_batch BEFORE DELETE ON MESSAGE WHEN OLD.ID = 200
				BEGIN SELECT RAISE(ABORT, 'interrupt next batch'); END`)
			require.NoError(t, err)
			batched := dedupeRepository{messageMode: mode, batchRows: 2}
			_, err = batched.Deduplicate(ctx, db)
			require.ErrorContains(t, err, "interrupt next batch")

			// The first oversized group committed in full. Chunk 11 has
			// payload in the interrupted range and must still be present.
			var got []int64
			require.NoError(t, db.Select(&got, `SELECT CHUNK_ID FROM MESSAGE WHERE ID = 100 ORDER BY CHUNK_ID`))
			require.Equal(t, []int64{21, 23}, got)
			got = nil
			require.NoError(t, db.Select(&got, `SELECT CHUNK_ID FROM MESSAGE WHERE ID = 200 ORDER BY CHUNK_ID`))
			require.Equal(t, []int64{11, 12, 22}, got)
			var nextID int64
			require.NoError(t, db.Get(&nextID, "SELECT KEY1 FROM DEDUPE_CHECKPOINT"))
			require.EqualValues(t, 200, nextID)
			require.NoError(t, db.Close())

			db = testConnDSN(t, DSN(path, ModeRW))
			_, err = db.Exec("DROP TRIGGER fail_batch")
			require.NoError(t, err)
			wantRemaining := want
			wantRemaining.Messages -= 3
			wantRemaining.CanvasMessages--
			wantRemaining.Chunks -= 2
			remaining, err := batched.Preview(ctx, db)
			require.NoError(t, err)
			require.Equal(t, wantRemaining, remaining)

			stats, err := batched.Deduplicate(ctx, db)
			require.NoError(t, err)
			require.Equal(t, want, stats)
			for _, table := range append([]string{"CHUNK"}, PayloadTables...) {
				snapshot := func(conn *sqlx.DB) []map[string]any {
					t.Helper()
					rows, err := conn.Queryx("SELECT * FROM " + table + " ORDER BY rowid")
					require.NoError(t, err)
					defer rows.Close()
					var data []map[string]any
					for rows.Next() {
						row := make(map[string]any)
						require.NoError(t, rows.MapScan(row))
						// Fixtures inserted at different wall-clock times can
						// have different metadata timestamps.
						delete(row, "LOAD_DTTM")
						delete(row, "CREATED_AT")
						data = append(data, row)
					}
					require.NoError(t, rows.Err())
					return data
				}
				require.Equal(t, snapshot(whole), snapshot(db), "table %s", table)
			}
			var checkpoints int
			require.NoError(t, db.Get(&checkpoints, "SELECT COUNT(*) FROM DEDUPE_CHECKPOINT"))
			require.Zero(t, checkpoints)
		})
	}

	t.Run("restarts at earlier keys when the mode changes", func(t *testing.T) {
		db := testConn(t)
		populateCheckpointDedupeTest(t, db)
		insertChunkForTest(t, db, 18, 1, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 18, []byte(`{"text":"edited"}`))

		exact := dedupeRepository{messageMode: MessageDedupeExact, batchRows: 2}
		progress, err := exact.deduplicateBatch(t.Context(), db, "", exact.batchRows)
		require.NoError(t, err)
		require.NotEmpty(t, progress.owner)

		byKey := dedupeRepository{messageMode: MessageDedupeKey, batchRows: 2}
		want, err := byKey.Preview(t.Context(), db)
		require.NoError(t, err)
		got, err := byKey.Deduplicate(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, want, got)
		var chunks []int64
		require.NoError(t, db.Select(&chunks, `SELECT CHUNK_ID FROM MESSAGE WHERE ID = 100 ORDER BY CHUNK_ID`))
		require.Equal(t, []int64{21, 23}, chunks)
	})

	t.Run("restarts at older keys after an archive write", func(t *testing.T) {
		db := testConn(t)
		populateCheckpointDedupeTest(t, db)
		repo := dedupeRepository{messageMode: MessageDedupeExact, batchRows: 2}
		progress, err := repo.deduplicateBatch(t.Context(), db, "", repo.batchRows)
		require.NoError(t, err)
		require.NotEmpty(t, progress.owner)

		// Normal archive writers insert payload and its chunk together. A
		// new copy below the saved cursor needs to be deduplicated again.
		tx, err := db.Beginx()
		require.NoError(t, err)
		_, err = tx.Exec(`INSERT INTO CHUNK (ID, SESSION_ID, TYPE_ID, UNIX_TS) VALUES (29, 2, 0, 0)`)
		require.NoError(t, err)
		_, err = tx.Exec(`INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA)
			SELECT ID, 29, CHANNEL_ID, TS, IDX, DATA FROM MESSAGE WHERE ID = 100 AND CHUNK_ID = 21`)
		require.NoError(t, err)
		require.NoError(t, tx.Commit())

		_, err = repo.deduplicateBatch(t.Context(), db, progress.owner, repo.batchRows)
		require.ErrorContains(t, err, "checkpoint changed")
		want, err := repo.Preview(t.Context(), db)
		require.NoError(t, err)
		got, err := repo.Deduplicate(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, want, got)
		var chunks []int64
		require.NoError(t, db.Select(&chunks, `SELECT CHUNK_ID FROM MESSAGE WHERE ID = 100 ORDER BY CHUNK_ID`))
		require.Equal(t, []int64{23, 29}, chunks)
	})

	t.Run("checks for a competing dedupe between batches", func(t *testing.T) {
		db := testConn(t)
		populateCheckpointDedupeTest(t, db)
		repo := dedupeRepository{messageMode: MessageDedupeExact, batchRows: 2}
		want, err := repo.Preview(t.Context(), db)
		require.NoError(t, err)
		first, err := repo.deduplicateBatch(t.Context(), db, "", repo.batchRows)
		require.NoError(t, err)
		require.NotEmpty(t, first.owner)
		second, err := repo.deduplicateBatch(t.Context(), db, "", repo.batchRows)
		require.NoError(t, err)
		require.NotEmpty(t, second.owner)
		require.NotEqual(t, first.owner, second.owner)
		_, err = repo.deduplicateBatch(t.Context(), db, first.owner, repo.batchRows)
		require.ErrorContains(t, err, "checkpoint changed")
		got, err := repo.Deduplicate(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	for _, mode := range []MessageDedupeMode{MessageDedupeExact, MessageDedupeKey} {
		t.Run("prefers finished sessions/"+string(mode), func(t *testing.T) {
			db := testConn(t)
			ctx := t.Context()
			repo := NewDedupeRepository(WithMessageDedupeMode(mode))

			insertSessionForTest(t, db, 1, true, nil)
			insertSessionForTest(t, db, 2, true, nil)
			insertSessionForTest(t, db, 3, false, nil)
			insertSessionForTest(t, db, 4, false, nil)
			for _, row := range []struct {
				session, chunk, message int64
				payload                 string
			}{
				{1, 11, 100, `{"text":"repeated"}`},
				{2, 21, 100, `{"text":"repeated"}`},
				{3, 31, 100, `{"text":"repeated"}`},
				{3, 32, 200, `{"text":"unfinished only"}`},
				{4, 42, 200, `{"text":"unfinished only"}`},
				{1, 12, 300, `{"text":"first"}`},
				{2, 22, 300, `{"text":"edited"}`},
				{3, 33, 300, `{"text":"edited again"}`},
			} {
				insertChunkForTest(t, db, row.chunk, row.session, chunk.CMessages)
				insertMessageWithChunkForTest(t, db, row.message, row.chunk, []byte(row.payload))
			}

			wantStats := DedupeStats{Messages: 3, Chunks: 3}
			wantSurvivors := []int64{12, 21, 22, 33, 42}
			wantAfterCleanup := []int64{12, 21, 22}
			if mode == MessageDedupeKey {
				wantStats = DedupeStats{Messages: 5, Chunks: 5}
				wantSurvivors = []int64{21, 22, 42}
				wantAfterCleanup = []int64{21, 22}
			}
			preview, err := repo.Preview(ctx, db)
			require.NoError(t, err)
			require.Equal(t, wantStats, preview)
			result, err := repo.Deduplicate(ctx, db)
			require.NoError(t, err)
			require.Equal(t, preview, result)
			var survivors []int64
			require.NoError(t, db.Select(&survivors, "SELECT CHUNK_ID FROM MESSAGE ORDER BY CHUNK_ID"))
			assert.Equal(t, wantSurvivors, survivors)

			cleanup, err := NewCleanupRepository().CleanupUnfinishedSessions(ctx, db)
			require.NoError(t, err)
			assert.EqualValues(t, 2, cleanup.SessionsRemoved)
			assert.EqualValues(t, len(wantSurvivors)-len(wantAfterCleanup), cleanup.PayloadRemoved["MESSAGE"])
			require.NoError(t, db.Select(&survivors, "SELECT CHUNK_ID FROM MESSAGE ORDER BY CHUNK_ID"))
			assert.Equal(t, wantAfterCleanup, survivors)
		})
		t.Run("skewed/"+string(mode), func(t *testing.T) {
			db := skewedDedupeFixture(t, 2000)
			expected := expectedDedupeMessages(t, db, mode)
			repo := NewDedupeRepository(WithMessageDedupeMode(mode))
			preview, err := repo.Preview(t.Context(), db)
			require.NoError(t, err)
			got, err := repo.Deduplicate(t.Context(), db)
			require.NoError(t, err)
			require.Equal(t, preview, got)
			var actual []string
			require.NoError(t, db.Select(&actual, `SELECT CHANNEL_ID || ':' || ID || ':' || CHUNK_ID || ':' || hex(DATA) FROM MESSAGE ORDER BY CHANNEL_ID, ID, CHUNK_ID`))
			require.Equal(t, expected, actual)
			verifyMessageChunkInChannelForTest(t, db, 0, "C-hot", 21)
			var count int
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM CHUNK WHERE ID=1`))
			require.Zero(t, count) // fully duplicated page
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM CHUNK WHERE ID=2`))
			require.Equal(t, 1, count) // partially duplicated page
			verifyChannelUserCountForTest(t, db, 500)
			require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM CHANNEL_USER WHERE CHUNK_ID=26`))
			require.Equal(t, 500, count)
			rows, err := db.Query(`PRAGMA foreign_key_check`)
			require.NoError(t, err)
			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
			rows.Close()
			again, err := repo.Deduplicate(t.Context(), db)
			require.NoError(t, err)
			require.Equal(t, DedupeStats{}, again)
			again, err = repo.Preview(t.Context(), db)
			require.NoError(t, err)
			require.Equal(t, DedupeStats{}, again)
		})
	}
	t.Run("rolls back earlier entities on deletion error", func(t *testing.T) {
		db := skewedDedupeFixture(t, 2000)
		before := expectedDedupeMessages(t, db, MessageDedupeExact)
		insertChunkForTest(t, db, 100, 1, chunk.CFiles)
		insertChunkForTest(t, db, 101, 1, chunk.CFiles)
		for _, id := range []int64{100, 101} {
			insertFileWithChunkForTest(t, db, "F1", id, "C-hot", nil, nil, []byte(`{}`))
		}
		_, err := db.Exec(`CREATE TRIGGER fail_file_delete BEFORE DELETE ON FILE BEGIN SELECT RAISE(ABORT, 'test failure'); END`)
		require.NoError(t, err)
		_, err = NewDedupeRepository().Deduplicate(t.Context(), db)
		require.ErrorContains(t, err, "test failure")
		require.Equal(t, before, expectedDedupeMessages(t, db, MessageDedupeExact))
		verifyChunkCountForTest(t, db, 28)
		verifyChannelUserCountForTest(t, db, 2000)
	})
	t.Run("cancelled", func(t *testing.T) {
		db := testConn(t)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := NewDedupeRepository().Deduplicate(ctx, db)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("deduplicates all supported entities and prunes duplicate-only chunks", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

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
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", nil, nil, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C001", nil, nil, []byte(`{"name":"same"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
			Messages:     1,
			Users:        1,
			Channels:     1,
			ChannelUsers: 1,
			Files:        1,
			Chunks:       5,
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
		insertSessionForTest(t, db, 2, true, new(int64(1)))

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
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", nil, nil, []byte(`{"name":"old"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C001", nil, nil, []byte(`{"name":"new"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
			ChannelUsers: 1,
			Chunks:       1,
		}, result)

		verifyChunkCountForTest(t, db, 9)
		verifyMessageCountForTest(t, db, 2)
		verifyUserCountForTest(t, db, 2)
		verifyChannelCountForTest(t, db, 2)
		verifyChannelUserCountForTest(t, db, 1)
		verifyFileCountForTest(t, db, 2)
	})

	t.Run("exact mode preserves volatile block id payloads", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"dRF"}]}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"0Xuo9"}]}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{}, result)
		verifyMessageCountForTest(t, db, 2)
		verifyChunkCountForTest(t, db, 2)
	})

	t.Run("message key mode deduplicates volatile block id payloads", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository(WithMessageDedupeMode(MessageDedupeKey))

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageWithChunkForTest(t, db, 100, 11, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"dRF"}]}`))
		insertMessageWithChunkForTest(t, db, 100, 21, []byte(`{"text":"same","blocks":[{"type":"rich_text","block_id":"0Xuo9"}]}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
			Messages: 1,
			Chunks:   1,
		}, result)
		verifyMessageCountForTest(t, db, 1)
		verifyChunkCountForTest(t, db, 1)
		verifyMessageChunkForTest(t, db, 100, 21)
	})

	t.Run("message key mode preserves same timestamp in different channels", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository(WithMessageDedupeMode(MessageDedupeKey))

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 11, 1, chunk.CMessages)
		insertChunkForTest(t, db, 21, 2, chunk.CMessages)
		insertMessageInChannelWithChunkForTest(t, db, 100, 11, "C001", []byte(`{"text":"same"}`))
		insertMessageInChannelWithChunkForTest(t, db, 100, 21, "C002", []byte(`{"text":"same"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{}, result)
		verifyMessageCountForTest(t, db, 2)
		verifyMessageChunkInChannelForTest(t, db, 100, "C001", 11)
		verifyMessageChunkInChannelForTest(t, db, 100, "C002", 21)
	})

	t.Run("preserves reshared files across different attachment contexts", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", new(int64(100)), nil, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C002", new(int64(200)), nil, []byte(`{"name":"same"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{}, result)

		verifyChunkCountForTest(t, db, 2)
		verifyFileCountForTest(t, db, 2)
	})

	t.Run("deduplicates files with matching nullable attachment context", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

		insertChunkForTest(t, db, 15, 1, chunk.CFiles)
		insertChunkForTest(t, db, 25, 2, chunk.CFiles)
		insertFileWithChunkForTest(t, db, "F100", 15, "C001", nil, nil, []byte(`{"name":"same"}`))
		insertFileWithChunkForTest(t, db, "F100", 25, "C001", nil, nil, []byte(`{"name":"same"}`))

		result, err := repo.Deduplicate(ctx, db)
		require.NoError(t, err)
		assert.Equal(t, DedupeStats{
			Files:  1,
			Chunks: 1,
		}, result)

		verifyChunkCountForTest(t, db, 1)
		verifyFileCountForTest(t, db, 1)
		verifyFileChunkForTest(t, db, "F100", 25)
	})

	t.Run("keeps chunks that still contain non-duplicate rows", func(t *testing.T) {
		db := testConn(t)
		ctx := context.Background()
		repo := NewDedupeRepository()

		insertSessionForTest(t, db, 1, true, nil)
		insertSessionForTest(t, db, 2, true, new(int64(1)))

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
		assert.Equal(t, DedupeStats{
			Messages:     1,
			Users:        1,
			ChannelUsers: 1,
			Chunks:       3,
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
		insertSessionForTest(t, db, 2, true, new(int64(1)))
		insertSessionForTest(t, db, 3, true, new(int64(2)))

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
		assert.Equal(t, DedupeStats{
			Messages:     2,
			Users:        2,
			ChannelUsers: 2,
			Chunks:       6,
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

func insertFileWithChunkForTest(t *testing.T, db *sqlx.DB, fileID string, chunkID int64, channelID string, messageID, threadID *int64, data []byte) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO FILE (ID, CHUNK_ID, CHANNEL_ID, MESSAGE_ID, THREAD_ID, IDX, MODE, FILENAME, URL, DATA, SIZE)
		VALUES (?, ?, ?, ?, ?, 0, 'hosted', 'file.txt', 'https://example.com/file', ?, 1)`,
		fileID, chunkID, channelID, messageID, threadID, data)
	require.NoError(t, err)
}

func verifyMessageChunkForTest(t *testing.T, db *sqlx.DB, msgID, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM MESSAGE WHERE ID = ?", msgID).Scan(&chunkID)
	require.NoError(t, err)
	assert.Equal(t, expectedChunkID, chunkID)
}

func verifyMessageChunkInChannelForTest(t *testing.T, db *sqlx.DB, msgID int64, channelID string, expectedChunkID int64) {
	t.Helper()
	var chunkID int64
	err := db.QueryRowxContext(context.Background(), "SELECT CHUNK_ID FROM MESSAGE WHERE ID = ? AND CHANNEL_ID = ?", msgID, channelID).Scan(&chunkID)
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

func Test_joinOnColumns(t *testing.T) {
	tests := []struct {
		name string
		cols []string
		want string
	}{
		{
			name: "multiple columns",
			cols: []string{"CHANNEL_ID", "USER_ID"},
			want: "T.CHANNEL_ID IS L.CHANNEL_ID AND T.USER_ID IS L.USER_ID",
		},
		{
			name: "one column",
			cols: []string{"MESSAGE_ID"},
			want: "T.MESSAGE_ID IS L.MESSAGE_ID",
		},
		{
			name: "no columns",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinOnColumns("T", "L", tt.cols))
		})
	}
}

func populateCheckpointDedupeTest(t *testing.T, db *sqlx.DB) {
	t.Helper()
	insertSessionForTest(t, db, 1, true, nil)
	insertSessionForTest(t, db, 2, true, nil)
	insertSessionForTest(t, db, 3, false, nil)
	for _, row := range []struct {
		sessionID, chunkID int64
		kind               chunk.ChunkType
	}{
		{1, 11, chunk.CMessages},
		{1, 12, chunk.CMessages},
		{2, 21, chunk.CThreadMessages},
		{2, 22, chunk.CThreadMessages},
		{3, 31, chunk.CMessages},
		{1, 13, chunk.CCanvasMessages},
		{2, 23, chunk.CCanvasThreadMessages},
	} {
		insertChunkForTest(t, db, row.chunkID, row.sessionID, row.kind)
	}
	for _, row := range []struct {
		id, chunkID int64
		payload     string
	}{
		{100, 11, `{"text":"same"}`},
		{100, 12, `{"text":"same"}`},
		{100, 21, `{"text":"same"}`},
		{100, 31, `{"text":"same"}`},
		{200, 11, `{"text":"original"}`},
		{200, 12, `{"text":"edited"}`},
		{200, 22, `{"text":"edited"}`},
		{300, 22, `{"text":"unique"}`},
		{100, 13, `{"text":"canvas"}`},
		{100, 23, `{"text":"canvas"}`},
	} {
		insertMessageWithChunkForTest(t, db, row.id, row.chunkID, []byte(row.payload))
	}
	payload := []byte(`{}`)
	for _, pair := range []struct {
		older, newer int64
		kind         chunk.ChunkType
	}{
		{101, 201, chunk.CUsers},
		{102, 202, chunk.CChannels},
		{103, 203, chunk.CChannelUsers},
		{104, 204, chunk.CFiles},
	} {
		insertChunkForTest(t, db, pair.older, 1, pair.kind)
		insertChunkForTest(t, db, pair.newer, 2, pair.kind)
		for _, chunkID := range []int64{pair.older, pair.newer} {
			for _, id := range []string{"1", "2", "3"} {
				switch pair.kind {
				case chunk.CUsers:
					insertUserWithChunkForTest(t, db, "U"+id, chunkID, "user", payload)
				case chunk.CChannels:
					insertChannelWithChunkForTest(t, db, "C"+id, chunkID, "channel", payload)
				case chunk.CChannelUsers:
					insertChannelUserWithChunkForTest(t, db, "C1", "U"+id, chunkID)
				case chunk.CFiles:
					insertFileWithChunkForTest(t, db, "F"+id, chunkID, "C1", nil, nil, payload)
				}
			}
		}
	}
}
