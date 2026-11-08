package diag

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

var cmdCleanup = &base.Command{
	UsageLine:  "slackdump tools cleanup [flags] <archive_directory>",
	Short:      "remove data from unfinished sessions",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Cleanup removes residual database rows that belong to sessions where
SESSION.FINISHED is false: the session itself, its chunks, and all payload
rows (messages, channels, files, users, etc.) stored in those chunks.

It also removes orphaned payload rows, i.e. rows whose chunk no longer
exists. Such rows could be left behind by earlier versions of this tool.

By default it only reports what would be removed. Pass the archive directory,
not the slackdump.sqlite file. Use -execute to perform the cleanup.

Cleanup does not shrink the database file. To reclaim the disk space, run
VACUUM on the database afterwards, for example:

    sqlite3 <archive_directory>/slackdump.sqlite VACUUM
`,
}

var cleanupFlags struct {
	execute bool
}

func init() {
	cmdCleanup.Run = runCleanup
	cmdCleanup.Flag.BoolVar(&cleanupFlags.execute, "execute", false, "actually remove unfinished session data")
}

func runCleanup(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	if cmd.Flag.NArg() != 1 {
		cmd.Flag.Usage()
		return nil
	}

	dbPath := cmd.Flag.Arg(0)
	conn, err := ensureDb(ctx, dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	repo := repository.NewCleanupRepository()

	sessionCount, err := repo.CountUnfinishedSessions(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting unfinished sessions: %w", err)
	}
	chunkCount, err := repo.CountUnfinishedChunks(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting unfinished chunks: %w", err)
	}
	payloadCount, err := repo.CountUnfinishedPayload(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting unfinished payload rows: %w", err)
	}
	orphanCount, err := repo.CountOrphanedPayload(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting orphaned payload rows: %w", err)
	}

	slog.DebugContext(ctx, "cleanup preview",
		"database", dbPath,
		"unfinished_sessions", sessionCount,
		"unfinished_chunks", chunkCount,
		"unfinished_payload_rows", payloadCount.Total(),
		"orphaned_payload_rows", orphanCount.Total(),
	)

	fmt.Printf("Unfinished sessions: %d\n", sessionCount)
	fmt.Printf("Chunks in unfinished sessions: %d\n", chunkCount)
	printTableCounts("Payload rows in unfinished sessions", payloadCount)
	printTableCounts("Orphaned payload rows", orphanCount)

	if !cleanupFlags.execute {
		if sessionCount > 0 || chunkCount > 0 || payloadCount.Total() > 0 || orphanCount.Total() > 0 {
			fmt.Println("\nRun with -execute to perform cleanup.")
		}
		return nil
	}

	result, err := repo.CleanupUnfinishedSessions(ctx, conn)
	if err != nil {
		return fmt.Errorf("cleanup unfinished sessions: %w", err)
	}
	orphansRemoved, err := repo.DeleteOrphanedPayload(ctx, conn)
	if err != nil {
		return fmt.Errorf("delete orphaned payload rows: %w", err)
	}
	fmt.Printf("\nRemoved sessions: %d\n", result.SessionsRemoved)
	fmt.Printf("Removed chunks: %d\n", result.ChunksRemoved)
	printTableCounts("Removed payload rows", result.PayloadRemoved)
	printTableCounts("Removed orphaned payload rows", orphansRemoved)

	if result.SessionsRemoved > 0 || result.PayloadRemoved.Total() > 0 || orphansRemoved.Total() > 0 {
		fmt.Println("\nTo reclaim disk space, run VACUUM on the database, i.e.:")
		fmt.Printf("    sqlite3 %q VACUUM\n", bootstrap.DatabasePath(dbPath))
	}
	return nil
}

// printTableCounts prints the total and the non-zero per-table counts in the
// stable order of [repository.PayloadTables].
func printTableCounts(title string, tc repository.TableCounts) {
	fmt.Printf("%s: %d\n", title, tc.Total())
	for _, table := range repository.PayloadTables {
		if n := tc[table]; n > 0 {
			fmt.Printf("  %-15s %d\n", table+":", n)
		}
	}
}
