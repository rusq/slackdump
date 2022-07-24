package app

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/fsadapter"
)

// Export performs the full export of slack workspace in slack export compatible
// format.
func Export(ctx context.Context, cfg Config, prov auth.Provider) error {
	ctx, task := trace.NewTask(ctx, "Export")
	defer task.End()

	if cfg.ExportName == "" {
		return errors.New("export directory or filename not specified")
	}

	sess, err := slackdump.NewWithOptions(ctx, prov, cfg.Options)
	if err != nil {
		return err
	}

	expCfg := export.Options{
		Oldest:       time.Time(cfg.Oldest),
		Latest:       time.Time(cfg.Latest),
		IncludeFiles: cfg.Options.DumpFiles,
		Logger:       cfg.Logger(),
		List:         cfg.Input.List,
	}
	fs, err := fsadapter.ForFilename(cfg.ExportName)
	if err != nil {
		cfg.Logger().Debugf("Export:  filesystem error: %s", err)
		return fmt.Errorf("failed to initialise the filesystem: %w", err)
	}
	defer func() {
		cfg.Logger().Debugf("Export:  closing file system")
		if err := fsadapter.Close(fs); err != nil {
			cfg.Logger().Printf("Export:  error closing filesystem")
		}
	}()

	cfg.Logger().Debugf("Export:  filesystem: %s", fs)
	cfg.Logger().Printf("Export:  staring export to: %s", fs)

	e := export.New(sess, fs, expCfg)
	if err := e.Run(ctx); err != nil {
		return err
	}

	return nil
}
