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
package diag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/redownload"
)

var cmdRedownload = &base.Command{
	UsageLine: "tools redownload [flags] <archive_dir>",
	Short:     "attempts to redownload missing files from the archive",
	Long: `# File redownload tool
Redownload tool scans the slackdump export, archive or dump directory,
validating the files.

If a file is missing or has zero length, it will be redownloaded from the Slack
API. The tool will not overwrite existing files, so it is safe to run it
multiple times.

**Please note:**

1. It requires you to have a valid authentication in the selected workspace.
2. Ensure that you have selected the correct workspace using "slackdump workspace select".
3. It only support directories.  ZIP files can not be updated. Unpack ZIP file
   to a directory before using this tool.
`,
	FlagMask:    cfg.OmitAll &^ (cfg.OmitAuthFlags | cfg.OmitWorkspaceFlag),
	Run:         runRedownload,
	PrintFlags:  true,
	RequireAuth: true,
}

type redownloadFlags struct {
	dryRun bool
}

var redlFlags redownloadFlags

func init() {
	cmdRedownload.Flag.BoolVar(&redlFlags.dryRun, "dry", redlFlags.dryRun, "estimate amd print the size and count of files to be downloaded, do not download anything")
	cmdRedownload.Flag.BoolVar(&redlFlags.dryRun, "estimate", redlFlags.dryRun, "alias for -dry")
}

func runRedownload(ctx context.Context, _ *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("expected exactly one argument")
	}
	dir := args[0]

	rd, err := redownload.New(ctx, dir, redownload.WithLogger(cfg.Log))
	if err != nil {
		return err
	}
	defer rd.Stop()

	var stats redownload.FileStats
	if redlFlags.dryRun {
		slog.WarnContext(ctx, "dry run/estimate mode, files will not be downloaded")
		defer func() {
			if err == nil {
				slog.WarnContext(ctx, "estimation only, actual numbers may differ")
			}
		}()
		stats, err = rd.Stats(ctx)
	} else {
		slog.InfoContext(ctx, "starting redownload")
		client, err := bootstrap.Slack(ctx)
		if err != nil {
			return fmt.Errorf("error creating slackdump session: %w", err)
		}
		stats, err = rd.Download(ctx, client)
	}
	if err != nil {
		return err
	}

	if stats.NumFiles == 0 {
		slog.InfoContext(ctx, "no missing files found")
	} else {
		slog.InfoContext(ctx, "estimated file download stats", stats.Attr())
	}

	return nil
}
