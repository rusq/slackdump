package app

import (
	"context"
	"errors"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/app/config"
)

// defExportType is the default file export type, if the DumpFiles is
// requested.
const defExportType = export.TStandard

// Export performs the full export of slack workspace in slack export compatible
// format.
func Export(ctx context.Context, cfg config.Params, prov auth.Provider) error {
	ctx, task := trace.NewTask(ctx, "Export")
	defer task.End()

	if cfg.ExportName == "" {
		return errors.New("export directory or filename not specified")
	}

	fs, err := fsadapter.New(cfg.ExportName)
	if err != nil {
		return err
	}
	defer fs.Close()

	sess, err := slackdump.New(ctx, prov, slackdump.WithFilesystem(fs), slackdump.WithLogger(dlog.FromContext(ctx)))
	if err != nil {
		return err
	}

	cfg.Logger().Printf("Export:  staring export to: %s", cfg.ExportName)

	e := export.New(sess, makeExportOptions(cfg))
	if err := e.Run(ctx); err != nil {
		return err
	}

	return nil
}

func makeExportOptions(cfg config.Params) export.Config {
	expCfg := export.Config{
		Oldest:      time.Time(cfg.Oldest),
		Latest:      time.Time(cfg.Latest),
		Logger:      cfg.Logger(),
		List:        cfg.Input.List,
		Type:        cfg.ExportType,
		ExportToken: cfg.ExportToken,
	}
	// if files requested, but the type is no-download, we need to switch
	// export type to the default export type, so that the files would
	// download.
	if cfg.DumpFiles && cfg.ExportType == export.TNoDownload {
		expCfg.Type = defExportType
	}
	return expCfg
}
