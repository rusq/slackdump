package convert

import (
	"context"
	"errors"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/convert"
	"github.com/rusq/slackdump/v2/logger"
)

var CmdConvert = &base.Command{
	Run:       runConvert,
	UsageLine: "slackdump convert [flags] <source>",
	Short:     "convert slackdump chunks to various formats",
	Long: `
# Convert Command

Convert slackdump Chunks (output of "record") to various formats.

By default it converts a directory with chunks to an archive or directory
in Slack Export format.
`,
	CustomFlags: false,
	FlagMask:    cfg.OmitAll & ^cfg.OmitDownloadFlag &^ cfg.OmitBaseLocFlag,
	PrintFlags:  true,
}

var storageType fileproc.StorageType

func init() {
	CmdConvert.Flag.Var(&storageType, "storage", "storage type")
}

func runConvert(ctx context.Context, cmd *base.Command, args []string) error {
	if len(args) < 1 {
		base.SetExitStatus(base.SInvalidParameters)
		return errors.New("source and destination are required")
	}
	src := args[0]
	trg := cfg.BaseLocation

	lg := logger.FromContext(ctx)
	lg.Printf("converting (chunk) %q to (export) %q", src, trg)
	cd, err := chunk.OpenDir(src)
	if err != nil {
		return err
	}
	fsa, err := fsadapter.New(trg)
	if err != nil {
		return err
	}
	defer fsa.Close()

	cvt := convert.NewChunkToExport(cd, fsa, convert.WithIncludeFiles(cfg.DumpFiles))
	if err := cvt.Convert(ctx); err != nil {
		return err
	}
	lg.Printf("done")

	return nil
}
