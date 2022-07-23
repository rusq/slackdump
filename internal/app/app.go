package app

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	OutputTypeJSON = "json"
	OutputTypeText = "text"
)

type App struct {
	sd *slackdump.Session

	prov auth.Provider
	cfg  Config
}

// New creates a new slackdump app. It inherits the logging from slack options
// in the Config.
func New(cfg Config, provider auth.Provider) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	app := &App{cfg: cfg, prov: provider}
	return app, nil
}

func (app *App) Run(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "app.Run")
	defer task.End()

	start := time.Now()

	var err error
	if app.cfg.ExportName != "" {
		err = app.Export(ctx, app.cfg.ExportName)
	} else {
		err = app.runDump(ctx)
	}
	if err != nil {
		return err
	}

	app.l().Printf("completed, time taken: %s", time.Since(start))
	return nil
}

// Close closes all open handles.
func (app *App) Close() error {
	return nil
}

func (app *App) runDump(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "runDump")
	defer task.End()

	dm, err := newDump(ctx, &app.cfg, app.prov)
	if err != nil {
		return err
	}

	if app.cfg.ListFlags.FlagsPresent() {
		err = dm.List(ctx)
	} else {
		var n int
		n, err = dm.Dump(ctx)
		app.l().Printf("dumped %d item(s)", n)
	}
	return err
}

func (app *App) l() logger.Interface {
	// inherit the logger from the slackdump options
	if app.cfg.Options.Logger == nil {
		app.cfg.Options.Logger = logger.Default
	}
	return app.cfg.Options.Logger
}

// td outputs the message to trace and logs a debug message.
func (app *App) td(ctx context.Context, category string, fmt string, a ...any) {
	app.l().Debugf(fmt, a...)
	trace.Logf(ctx, category, fmt, a...)
}

// td outputs the message to trace and logs a debug message.
func (app *App) tl(ctx context.Context, category string, fmt string, a ...any) {
	app.l().Printf(fmt, a...)
	trace.Logf(ctx, category, fmt, a...)
}
