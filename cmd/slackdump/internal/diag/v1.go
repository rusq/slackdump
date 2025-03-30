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
	"github.com/rusq/slackdump/v3/internal/source"
)

var cmdConvertV1 = &base.Command{
	UsageLine: "slackdump convertv1 [flags] <path>",
	Short:     "slackdump v1.0.x conversion utility",
	Long: `# Conversion utility for slackdump v1.0.x files

Slackdump v1.0.x are rare in the wild, but if you have one, you can use this
command to convert it to current dump format to be able to use it with the
viewer and other commands.`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll &^ cfg.OmitOutputFlag &^ cfg.OmitDownloadFlag,
	PrintFlags:  true,
	RequireAuth: false,
	HideWizard:  true,
	Run:         runV1,
}

func runV1(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("must provide a single path to v1.0.x dump")
	}
	path := args[0]

	output := cfg.StripZipExt(cfg.Output)

	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	wconn, si, err := bootstrap.Database(output, "v1")
	if err != nil {
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
		if err := convertFile(ctx, erc, filepath.Join(path, p), output); err != nil {
			return fmt.Errorf("failed to convert file %q: %w", p, err)
		}
		return nil
	}); err != nil {
		return err
	}

	slog.InfoContext(ctx, "v1.0.x dump converted successfully", "output", output, "took", time.Since(start).String())
	return nil
}

func convertFile(ctx context.Context, erc chunk.Encoder, path string, outputDir string) error {
	src, err := sdv1.NewSource(path)
	if err != nil {
		return fmt.Errorf("failed to create source: %w", err)
	}
	fsa := fsadapter.NewDirectory(outputDir)
	conv := convert.NewSourceEncoder(src, fsa, erc, convert.WithIncludeFiles(cfg.WithFiles), convert.WithLogger(cfg.Log), convert.WithIncludeAvatars(false), convert.WithTrgFileLoc(source.MattermostFilepath))
	if err := conv.Convert(ctx); err != nil {
		return fmt.Errorf("failed to convert source: %w", err)
	}
	return nil
}
