package app

import (
	"context"
	"errors"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/export"
)

// Export performs the full export of slack workspace in slack export compatible
// format.
func (app *App) Export(ctx context.Context, name string) error {
	ctx, task := trace.NewTask(ctx, "App.Export")
	defer task.End()

	if name == "" {
		return errors.New("export directory or filename not specified")
	}

	cfg := export.Options{
		Oldest:       time.Time(app.cfg.Oldest),
		Latest:       time.Time(app.cfg.Latest),
		IncludeFiles: app.cfg.Options.DumpFiles,
		Logger:       app.l(),
		List:         app.cfg.Input.List,
	}
	fs, err := fsadapter.ForFilename(name)
	if err != nil {
		app.td(ctx, "error", "filesystem: %s", err)
		return fmt.Errorf("failed to initialise the filesystem: %w", err)
	}
	defer func() {
		app.td(ctx, "info", "closing file system")
		if err := fsadapter.Close(fs); err != nil {
			app.tl(ctx, "error", "error closing filesystem")
		}
	}()
	app.td(ctx, "info", "filesystem: %s", fs)

	app.tl(ctx, "info", "staring export to: %s", fs)

	e := export.New(app.sd, fs, cfg)
	if err := e.Run(ctx); err != nil {
		return err
	}

	return nil
}
