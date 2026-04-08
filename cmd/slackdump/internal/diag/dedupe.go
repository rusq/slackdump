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
	Short:      "deduplicate overlap entities from resume runs",
	FlagMask:   cfg.OmitAll,
	PrintFlags: true,
	Long: `
Dedupe removes identical duplicate messages, users, channels, channel users,
and files created by resume look-back overlap. The latest copy of each
identical payload is kept. By default it only reports what would be removed.
Use -execute to perform deduplication.
`,
}

var dedupeFlags struct {
	execute bool
}

func init() {
	cmdDedupe.Run = runDedupe
	cmdDedupe.Flag.BoolVar(&dedupeFlags.execute, "execute", false, "actually remove duplicate entities")
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

	repo := repository.NewDedupeRepository()
	conn := src.Conn()

	counts, err := repo.Preview(ctx, conn)
	if err != nil {
		return fmt.Errorf("preview dedupe: %w", err)
	}

	slog.DebugContext(ctx, "dedupe preview",
		"database", dbPath,
		"duplicate_messages", counts.Messages,
		"duplicate_users", counts.Users,
		"duplicate_channels", counts.Channels,
		"duplicate_channel_users", counts.ChannelUsers,
		"duplicate_files", counts.Files,
		"prunable_chunks", counts.Chunks,
	)

	fmt.Printf("Duplicate messages: %d\n", counts.Messages)
	fmt.Printf("Duplicate users: %d\n", counts.Users)
	fmt.Printf("Duplicate channels: %d\n", counts.Channels)
	fmt.Printf("Duplicate channel users: %d\n", counts.ChannelUsers)
	fmt.Printf("Duplicate files: %d\n", counts.Files)
	fmt.Printf("Chunks to prune: %d\n", counts.Chunks)

	if !dedupeFlags.execute {
		if counts.Messages > 0 || counts.Users > 0 || counts.Channels > 0 || counts.ChannelUsers > 0 || counts.Files > 0 || counts.Chunks > 0 {
			fmt.Println("\nRun with -execute to perform dedupe.")
		}
		return nil
	}

	result, err := repo.Deduplicate(ctx, conn)
	if err != nil {
		return fmt.Errorf("deduplicate entities: %w", err)
	}

	fmt.Printf("\nRemoved messages: %d\n", result.MessagesRemoved)
	fmt.Printf("Removed users: %d\n", result.UsersRemoved)
	fmt.Printf("Removed channels: %d\n", result.ChannelsRemoved)
	fmt.Printf("Removed channel users: %d\n", result.ChannelUsersRemoved)
	fmt.Printf("Removed files: %d\n", result.FilesRemoved)
	fmt.Printf("Removed chunks: %d\n", result.ChunksRemoved)
	return nil
}
