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

package fileproc

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"

	"github.com/rusq/slackdump/v4/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v4/processor"
)

// DeduplicatingFileProcessor wraps a FileProcessor and skips downloading
// files that already exist in the database with the same ID and size.
type DeduplicatingFileProcessor struct {
	inner processor.Filer
	db    *sqlx.DB
	lg    *slog.Logger
	r     repository.FileRepository
}

// NewDeduplicatingFileProcessor creates a new file processor that skips
// downloading files that already exist in the database.
func NewDeduplicatingFileProcessor(inner processor.Filer, db *sqlx.DB, lg *slog.Logger) *DeduplicatingFileProcessor {
	if lg == nil {
		lg = slog.Default()
	}
	return &DeduplicatingFileProcessor{
		inner: inner,
		db:    db,
		lg:    lg,
		r:     repository.NewFileRepository(),
	}
}

// Files processes files, skipping those that already exist in the database
// with the same ID and size.
//
// Files are processed one-by-one rather than batched, because:
//   - Files arrive per-message, so we can't batch across all files (streaming)
//   - Each file download is independent and can start immediately
//   - The underlying Filer handles its own batching/parallelism if needed
//
// Note: This makes one DB query per file. For messages with multiple files,
// this could be optimized to batch the lookup (WHERE ID IN (...) AND SIZE IN (...)),
// but most messages have 0-2 files so the benefit is limited.
func (d *DeduplicatingFileProcessor) Files(ctx context.Context, channel *slack.Channel, parent slack.Message, ff []slack.File) error {

	for i := range ff {
		f := &ff[i]
		if !IsValid(f) {
			continue
		}

		// Check if file already exists with same ID and size
		existing, err := d.r.GetByIDAndSize(ctx, d.db, f.ID, int64(f.Size))
		if err != nil {
			d.lg.WarnContext(ctx, "error checking file existence", "error", err, "file_id", f.ID)
			// Continue with download on error
		}

		if existing != nil {
			d.lg.DebugContext(ctx, "skipping duplicate file", "file_id", f.ID, "size", f.Size)
			continue
		}

		// File doesn't exist or size differs - download it
		if err := d.inner.Files(ctx, channel, parent, []slack.File{*f}); err != nil {
			return err
		}
	}

	return nil
}

// Close stops the underlying downloader.
func (d *DeduplicatingFileProcessor) Close() error {
	return d.inner.Close()
}
