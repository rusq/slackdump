package diag

import (
	"context"
	"fmt"
	"log/slog"

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
SESSION.FINISHED is false. By default it only reports what would be removed.
Pass the archive directory, not the slackdump.sqlite file. Use -execute to
perform the cleanup.
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

	slog.DebugContext(ctx, "cleanup preview", "database", dbPath, "unfinished_sessions", sessionCount, "unfinished_chunks", chunkCount)

	fmt.Printf("Unfinished sessions: %d\n", sessionCount)
	fmt.Printf("Chunks in unfinished sessions: %d\n", chunkCount)

	if !cleanupFlags.execute {
		if sessionCount > 0 || chunkCount > 0 {
			fmt.Println("\nRun with -execute to perform cleanup.")
		}
		return nil
	}

	result, err := repo.CleanupUnfinishedSessions(ctx, conn)
	if err != nil {
		return fmt.Errorf("cleanup unfinished sessions: %w", err)
	}
	fmt.Printf("\nRemoved sessions: %d\n", result.SessionsRemoved)
	fmt.Printf("Removed chunks: %d\n", result.ChunksRemoved)
	return nil
}
