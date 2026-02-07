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
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/diag/sdv1"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v3/internal/convert"
	"github.com/rusq/slackdump/v3/source"
)

var cmdConvertV1 = &base.Command{
	UsageLine: "slackdump convertv1 [flags] <path>",
	Short:     "slackdump v1.0.x conversion utility",
	Long: `# Conversion utility for slackdump v1.0.x files

Slackdump v1.0.x are rare in the wild, but if you have one, you can use this
command to convert it to current dump format to be able to use it with the
viewer and other commands.`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll &^ cfg.OmitOutputFlag &^ cfg.OmitWithFilesFlag,
	PrintFlags:  true,
	RequireAuth: false,
	HideWizard:  true,
	Run:         runV1,
}

var v1Flags = struct {
	ignoreCopyErrors bool
}{
	ignoreCopyErrors: true,
}

func init() {
	cmdConvertV1.Flag.BoolVar(&v1Flags.ignoreCopyErrors, "i", v1Flags.ignoreCopyErrors, "ignore copy errors")
}

func runV1(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return fmt.Errorf("must provide a single path to v1.0.x dump")
	}
	path := args[0]

	output := cfg.StripZipExt(cfg.Output)

	if err := os.MkdirAll(output, 0o755); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	wconn, si, err := bootstrap.Database(output, "v1")
	if err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer wconn.Close()

	start := time.Now()
	if err := fs.WalkDir(os.DirFS(path), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if match, err := filepath.Match("[CDG]*.json", name); err != nil || !match {
			return nil
		}
		erc, err := dbase.New(ctx, wconn, si, dbase.WithVerbose(cfg.Verbose))
		if err != nil {
			return fmt.Errorf("failed to create new session: %w", err)
		}
		defer erc.Close()
		if err := convertFile(ctx, erc, filepath.Join(path, p), output, v1Flags.ignoreCopyErrors); err != nil {
			return fmt.Errorf("failed to convert file %q: %w", p, err)
		}
		return nil
	}); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}

	slog.InfoContext(ctx, "v1.0.x dump converted successfully", "output", output, "took", time.Since(start).String())
	return nil
}

func convertFile(ctx context.Context, erc chunk.Encoder, path string, outputDir string, ignoreCopyErr bool) error {
	src, err := sdv1.NewSource(path)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}
	fsa := fsadapter.NewDirectory(outputDir)
	conv := convert.NewSourceEncoder(src, fsa, erc, convert.WithIncludeFiles(cfg.WithFiles), convert.WithLogger(cfg.Log), convert.WithIncludeAvatars(false), convert.WithTrgFileLoc(source.MattermostFilepath), convert.WithIgnoreCopyErrors(ignoreCopyErr))
	if err := conv.Convert(ctx); err != nil {
		return fmt.Errorf("failed to convert source: %w", err)
	}
	return nil
}
