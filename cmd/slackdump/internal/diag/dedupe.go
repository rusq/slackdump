package diag

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

var cmdDedupe = &base.Command{
	UsageLine:  "slackdump tools dedupe [flags] <database_path>",
	Short:      "deduplicate overlap messages from resume runs",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Dedupe removes identical duplicate messages created by resume look-back overlap.
The latest copy of each identical message payload is kept. By default it only
reports what would be removed. Use -execute to perform deduplication.
`,
}

var dedupeFlags struct {
	execute bool
}

func init() {
	cmdDedupe.Run = runDedupe
	cmdDedupe.Flag.BoolVar(&dedupeFlags.execute, "execute", false, "actually remove duplicate messages")
}

func runDedupe(ctx context.Context, cmd *base.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	if cmd.Flag.NArg() != 1 {
		cmd.Flag.Usage()
		return nil
	}

	dbPath := cmd.Flag.Arg(0)
	src, err := dbase.OpenRW(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("opening database %q: %w", dbPath, err)
	}
	defer src.Close()

	repo := repository.NewMessageDedupeRepository()
	conn := src.Conn()

	messageCount, err := repo.CountDuplicateMessages(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting duplicate messages: %w", err)
	}
	chunkCount, err := repo.CountPrunableMessageChunks(ctx, conn)
	if err != nil {
		return fmt.Errorf("counting prunable chunks: %w", err)
	}

	slog.DebugContext(ctx, "dedupe preview", "database", dbPath, "duplicate_messages", messageCount, "prunable_chunks", chunkCount)

	fmt.Printf("Duplicate messages: %d\n", messageCount)
	fmt.Printf("Message chunks to prune: %d\n", chunkCount)

	if !dedupeFlags.execute {
		if messageCount > 0 || chunkCount > 0 {
			fmt.Println("\nRun with -execute to perform dedupe.")
		}
		return nil
	}

	result, err := repo.DeduplicateMessages(ctx, conn)
	if err != nil {
		return fmt.Errorf("deduplicate messages: %w", err)
	}

	fmt.Printf("\nRemoved messages: %d\n", result.MessagesRemoved)
	fmt.Printf("Removed chunks: %d\n", result.ChunksRemoved)
	return nil
}
