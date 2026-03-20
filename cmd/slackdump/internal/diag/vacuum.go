package diag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v4/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

// extractDBPath extracts the database path from args, allowing it to appear
// anywhere in the argument list (before, between, or after flags).
func extractDBPath(args []string) (string, []string) {
	dbPath := ""
	flagArgs := []string{}
	dbPathFound := false

	for _, arg := range args {
		if dbPathFound {
			flagArgs = append(flagArgs, arg)
		} else if !strings.HasPrefix(arg, "-") {
			dbPath = arg
			dbPathFound = true
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}
	return dbPath, flagArgs
}

var cmdVacuum = &base.Command{
	UsageLine:  "slackdump tools vacuum [flags] [database_path]",
	Short:      "vacuum the database",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Vacuum removes unreferenced chunks and duplicate entries from the database.

Duplicate users, messages, and files are removed only if the DATA is identical,
preserving edit history. Use --users, --messages, --files, or --chunks flags
to target specific cleanup operations.
`,
}

var vacuumFlags struct {
	execute  bool
	debug    bool
	chunks   bool
	users    bool
	messages bool
	files    bool
}

func init() {
	cmdVacuum.Run = runVacuum
	cmdVacuum.Flag.BoolVar(&vacuumFlags.execute, "execute", false, "actually perform the vacuum")
	cmdVacuum.Flag.BoolVar(&vacuumFlags.debug, "debug", false, "verbose output")
	cmdVacuum.Flag.BoolVar(&vacuumFlags.chunks, "chunks", false, "only remove unreferenced chunks")
	cmdVacuum.Flag.BoolVar(&vacuumFlags.users, "users", false, "only remove duplicate users")
	cmdVacuum.Flag.BoolVar(&vacuumFlags.messages, "messages", false, "only remove duplicate messages")
	cmdVacuum.Flag.BoolVar(&vacuumFlags.files, "files", false, "only remove duplicate files")
}

func runVacuum(ctx context.Context, cmd *base.Command, args []string) error {
	dbPath, flagArgs := extractDBPath(args)

	if err := cmd.Flag.Parse(flagArgs); err != nil {
		return err
	}

	if dbPath == "" {
		cmd.Flag.Usage()
		return nil
	}

	src, err := dbase.OpenRW(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("opening database %q: %w", dbPath, err)
	}
	defer src.Close()

	slog.DebugContext(ctx, "analyzing database", "path", dbPath)

	runAll := !vacuumFlags.chunks && !vacuumFlags.users && !vacuumFlags.messages && !vacuumFlags.files

	slog.DebugContext(ctx, "vacuum mode", "all", runAll, "users", vacuumFlags.users, "messages", vacuumFlags.messages, "chunks", vacuumFlags.chunks, "files", vacuumFlags.files)

	vr := repository.NewVacuumRepository()
	conn := src.Conn()

	var totalCount int64
	var totalRemoved int64

	if runAll || vacuumFlags.users {
		count, err := vr.CountDuplicateUsers(ctx, conn)
		if err != nil {
			return fmt.Errorf("counting duplicate users: %w", err)
		}
		fmt.Printf("Duplicate users: %d\n", count)
		totalCount += count

		if vacuumFlags.execute {
			deleted, err := vr.DeduplicateUsers(ctx, conn)
			if err != nil {
				return fmt.Errorf("deduplicating users: %w", err)
			}
			totalRemoved += deleted
		}
	}

	if runAll || vacuumFlags.messages {
		count, err := vr.CountDuplicateMessages(ctx, conn)
		if err != nil {
			return fmt.Errorf("counting duplicate messages: %w", err)
		}
		fmt.Printf("Duplicate messages: %d\n", count)
		totalCount += count

		if vacuumFlags.execute {
			deleted, err := vr.DeduplicateMessages(ctx, conn)
			if err != nil {
				return fmt.Errorf("deduplicating messages: %w", err)
			}
			totalRemoved += deleted
		}
	}

	if runAll || vacuumFlags.files {
		count, err := vr.CountDuplicateFiles(ctx, conn)
		if err != nil {
			return fmt.Errorf("counting duplicate files: %w", err)
		}
		fmt.Printf("Duplicate files: %d\n", count)
		totalCount += count

		if vacuumFlags.execute {
			deleted, err := vr.DeduplicateFiles(ctx, conn)
			if err != nil {
				return fmt.Errorf("deduplicating files: %w", err)
			}
			totalRemoved += deleted
		}
	}

	if runAll || vacuumFlags.chunks {
		// Chunk pruning must run last. Deleting duplicate users and messages
		// can leave their parent chunks unreferenced, so we identify and remove
		// those after deduplication is complete.
		count, err := vr.CountUnreferencedChunks(ctx, conn)
		if err != nil {
			return fmt.Errorf("counting unreferenced chunks: %w", err)
		}
		fmt.Printf("Unreferenced chunks: %d\n", count)
		totalCount += count

		if vacuumFlags.execute {
			deleted, err := vr.PruneUnreferencedChunks(ctx, conn)
			if err != nil {
				return fmt.Errorf("vacuum: %w", err)
			}
			totalRemoved += deleted
		}
	}

	if vacuumFlags.execute {
		fmt.Printf("\nRemoved: %d\n", totalRemoved)
		return nil
	}
	fmt.Printf("\nTotal to remove: %d\n", totalCount)
	if totalCount > 0 {
		fmt.Println("Run with -execute to perform vacuum.")
	}
	return nil
}
