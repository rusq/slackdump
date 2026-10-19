package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// skewedDedupeFixture has 90% of messages in one channel, alternating history
// and thread pages, a fully duplicated page, a partial overlap, an edit, and
// the same timestamp in another channel. Membership pages have 75% overlap.
// It deliberately has no ANALYZE statistics.
func skewedDedupeFixture(tb testing.TB, n int) *sqlx.DB {
	tb.Helper()
	db, err := sqlx.Open(Driver, ":memory:")
	require.NoError(tb, err)
	db.SetMaxOpenConns(1)
	tb.Cleanup(func() { db.Close() })
	_, err = db.Exec(`PRAGMA foreign_keys=ON`)
	require.NoError(tb, err)
	require.NoError(tb, Migrate(context.Background(), db.DB, false))
	tx, err := db.Beginx()
	require.NoError(tb, err)
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO SESSION (ID, MODE) VALUES (1, 'archive')`)
	require.NoError(tb, err)
	pages := n / 100
	for page := 0; page < pages+2; page++ {
		_, err = tx.Exec(`INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (?, 0, 1, ?)`, page+1, page%2)
		require.NoError(tb, err)
	}
	stmt, err := tx.Preparex(`INSERT INTO MESSAGE (ID, CHUNK_ID, CHANNEL_ID, TS, IDX, DATA) VALUES (?, ?, ?, ?, 0, ?)`)
	require.NoError(tb, err)
	defer stmt.Close()
	for i := 0; i < n; i++ {
		channel := "C-hot"
		if i >= n*9/10 {
			channel = fmt.Sprintf("C-%d", i%10)
		}
		payload := []byte(fmt.Sprintf(`{"id":%d,"text":"%s"}`, i, strings.Repeat("x", 768)))
		_, err = stmt.Exec(i, i/100+1, channel, fmt.Sprint(i), payload)
		require.NoError(tb, err)
		if i < 126 {
			if i == 125 {
				payload = []byte(`{"text":"edited"}`)
			}
			_, err = stmt.Exec(i, pages+1, channel, fmt.Sprint(i), payload)
			require.NoError(tb, err)
		}
	}
	_, err = stmt.Exec(0, pages+2, "C-other", "0", []byte(`{"text":"other channel"}`))
	require.NoError(tb, err)
	for page := 0; page < 4; page++ {
		_, err = tx.Exec(`INSERT INTO CHUNK (ID, UNIX_TS, SESSION_ID, TYPE_ID) VALUES (?, 0, 1, 7)`, pages+3+page)
		require.NoError(tb, err)
		for user := 0; user < n/4; user++ {
			_, err = tx.Exec(`INSERT INTO CHANNEL_USER (CHANNEL_ID, USER_ID, CHUNK_ID, IDX) VALUES ('C-hot', ?, ?, 0)`, fmt.Sprintf("U%d", user), pages+3+page)
			require.NoError(tb, err)
		}
	}
	require.NoError(tb, tx.Commit())
	return db
}

// expectedDedupeMessages uses GROUP BY alone, independently of the candidate
// join, to describe exactly which copies and payload variants must survive.
func expectedDedupeMessages(t *testing.T, db *sqlx.DB, mode MessageDedupeMode) []string {
	t.Helper()
	group := "CHANNEL_ID, ID"
	if mode == MessageDedupeExact {
		group += ", DATA"
	}
	var rows []string
	require.NoError(t, db.Select(&rows, `SELECT CHANNEL_ID || ':' || ID || ':' || MAX(CHUNK_ID) || ':' || hex(DATA) FROM MESSAGE GROUP BY `+group+` ORDER BY CHANNEL_ID, ID, MAX(CHUNK_ID)`))
	return rows
}

func Test_withDuplicateRows(t *testing.T) {
	for _, mode := range []MessageDedupeMode{MessageDedupeExact, MessageDedupeKey} {
		t.Run(string(mode), func(t *testing.T) {
			db := skewedDedupeFixture(t, 2000)
			var stats int
			require.NoError(t, db.Get(&stats, `SELECT COUNT(*) FROM sqlite_master WHERE name='sqlite_stat1'`))
			require.Zero(t, stats)
			repo := dedupeRepository{messageMode: mode}
			for _, entity := range repo.entities() {
				if entity.table != "MESSAGE" && entity.table != "CHANNEL_USER" {
					continue
				}
				t.Run(entity.etype.String(), func(t *testing.T) {
					for _, query := range []struct {
						name, final string
						pruning     bool
					}{
						{"count", `SELECT COUNT(1) FROM duplicates`, false},
						{"pruning", buildPrunableChunksSelect(entity), true},
						{"delete", buildDeleteDuplicatesStmt(entity), false},
					} {
						t.Run(query.name, func(t *testing.T) {
							args := chunkTypeArgs(entity.chunkTypes)
							if query.pruning {
								args = append(args, chunkTypeArgs(entity.chunkTypes)...)
							}
							rows, err := db.Query(`EXPLAIN QUERY PLAN `+withDuplicateRows(entity, query.final), args...)
							require.NoError(t, err)
							defer rows.Close()
							var details []string
							identityLookup := false
							for rows.Next() {
								var id, parent, unused int
								var detail string
								require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
								details = append(details, detail)
								// MESSAGE_I1 scans while building latest are legitimate. Reject only
								// candidate searches that constrain channel/chunk without identity.
								if strings.Contains(detail, "SEARCH T USING") && strings.Contains(detail, "CHANNEL_ID=?") {
									if entity.table == "MESSAGE" {
										require.Contains(t, detail, " ID=?", "%s", detail)
										identityLookup = true
									} else if strings.Contains(detail, "USER_ID=?") {
										identityLookup = true
									}
								}
							}
							require.NoError(t, rows.Err())
							// DELETE can instead scan T and probe latest by its grouped identity.
							if query.name != "delete" {
								require.True(t, identityLookup, "missing identity lookup: %v", details)
							}
							t.Log(strings.Join(details, "\n"))
						})
					}
				})
			}
		})
	}
}

func BenchmarkDedupeRepository_Preview(b *testing.B) {
	for _, n := range []int{2000, 4000, 8000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			db := skewedDedupeFixture(b, n)
			repo := NewDedupeRepository()
			b.ResetTimer()
			for b.Loop() {
				_, err := repo.Preview(context.Background(), db)
				require.NoError(b, err)
			}
		})
	}
}

func BenchmarkDedupeRepository_Deduplicate(b *testing.B) {
	for _, n := range []int{2000, 4000, 8000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			// Restore the same populated state outside the timed region on every run.
			// A benchmark of repeated no-op cleanup would hide duplicate removal cost.
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				db := skewedDedupeFixture(b, n)
				b.StartTimer()
				_, err := NewDedupeRepository().Deduplicate(context.Background(), db)
				b.StopTimer()
				require.NoError(b, err)
				require.NoError(b, db.Close())
			}
		})
	}
}
