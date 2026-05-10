package dedupe

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
)

type Options struct {
	Execute  bool
	Report   io.Writer
	Database string
}

type Result struct {
	Counts  repository.DedupeCounts
	Removed repository.DedupeResult
}

var newRepo = repository.NewDedupeRepository

func Run(ctx context.Context, db *sqlx.DB, opts Options) (Result, error) {
	repo := newRepo()

	counts, err := repo.Preview(ctx, db)
	if err != nil {
		return Result{}, fmt.Errorf("preview dedupe: %w", err)
	}

	slog.DebugContext(ctx, "dedupe preview",
		"database", opts.Database,
		"duplicate_messages", counts.Messages,
		"duplicate_users", counts.Users,
		"duplicate_channels", counts.Channels,
		"duplicate_channel_users", counts.ChannelUsers,
		"duplicate_files", counts.Files,
		"prunable_chunks", counts.Chunks,
	)

	if opts.Report != nil {
		fmt.Fprintf(opts.Report, "Duplicate messages: %d\n", counts.Messages)
		fmt.Fprintf(opts.Report, "Duplicate users: %d\n", counts.Users)
		fmt.Fprintf(opts.Report, "Duplicate channels: %d\n", counts.Channels)
		fmt.Fprintf(opts.Report, "Duplicate channel users: %d\n", counts.ChannelUsers)
		fmt.Fprintf(opts.Report, "Duplicate files: %d\n", counts.Files)
		fmt.Fprintf(opts.Report, "Chunks to prune: %d\n", counts.Chunks)
	}

	result := Result{Counts: counts}
	if !opts.Execute {
		if opts.Report != nil &&
			(counts.Messages > 0 || counts.Users > 0 || counts.Channels > 0 || counts.ChannelUsers > 0 || counts.Files > 0 || counts.Chunks > 0) {
			fmt.Fprintln(opts.Report, "\nRun with -execute to perform dedupe.")
		}
		return result, nil
	}

	removed, err := repo.Deduplicate(ctx, db)
	if err != nil {
		return Result{}, fmt.Errorf("deduplicate entities: %w", err)
	}
	result.Removed = removed

	slog.InfoContext(ctx, "dedupe execute",
		"database", opts.Database,
		"removed_messages", removed.MessagesRemoved,
		"removed_users", removed.UsersRemoved,
		"removed_channels", removed.ChannelsRemoved,
		"removed_channel_users", removed.ChannelUsersRemoved,
		"removed_files", removed.FilesRemoved,
		"removed_chunks", removed.ChunksRemoved,
	)

	if opts.Report != nil {
		fmt.Fprintf(opts.Report, "\nRemoved messages: %d\n", removed.MessagesRemoved)
		fmt.Fprintf(opts.Report, "Removed users: %d\n", removed.UsersRemoved)
		fmt.Fprintf(opts.Report, "Removed channels: %d\n", removed.ChannelsRemoved)
		fmt.Fprintf(opts.Report, "Removed channel users: %d\n", removed.ChannelUsersRemoved)
		fmt.Fprintf(opts.Report, "Removed files: %d\n", removed.FilesRemoved)
		fmt.Fprintf(opts.Report, "Removed chunks: %d\n", removed.ChunksRemoved)
	}
	return result, nil
}
