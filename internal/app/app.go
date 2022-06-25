package app

import (
	"context"
	"html/template"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/logger"
)

const (
	OutputTypeJSON = "json"
	OutputTypeText = "text"
)

type App struct {
	sd   *slackdump.Session
	tmpl *template.Template
	fs   fsadapter.FS

	prov auth.Provider
	cfg  Config
}

// New creates a new slackdump app. It inherits the logging from slack optiond
// in the Config.
func New(cfg Config, provider auth.Provider) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	tmpl, err := cfg.compileTemplates()
	if err != nil {
		return nil, err
	}
	fs, err := fsadapter.ForFilename(cfg.Output.Base)
	if err != nil {
		return nil, err
	}
	app := &App{cfg: cfg, prov: provider, tmpl: tmpl, fs: fs}
	return app, nil
}

func (app *App) Run(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "app.Run")
	defer task.End()

	if app.cfg.ExportName != "" {
		app.l().Debug("export mode ON")
	}

	if err := app.initSlackdump(ctx); err != nil {
		return err
	}

	start := time.Now()
	var err error
	switch {
	case app.cfg.ListFlags.FlagsPresent():
		err = app.runListEntities(ctx)
	case app.cfg.ExportName != "":
		err = app.runExport(ctx)
	default:
		err = app.runDump(ctx)
	}

	if err != nil {
		trace.Log(ctx, "error", err.Error())
		return err
	}

	app.l().Printf("completed, time taken: %s", time.Since(start))
	return nil
}

// Close closes all open handles.
func (app *App) Close() error {
	return fsadapter.Close(app.fs)
}

// initSlackdump initialises the slack dumper app.
func (app *App) initSlackdump(ctx context.Context) error {
	sd, err := slackdump.NewWithOptions(
		ctx,
		app.prov,
		app.cfg.Options,
	)
	if err != nil {
		return err
	}
	app.sd = sd
	app.sd.SetFS(app.fs)
	return nil
}

func (app *App) runListEntities(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "runListEntities")
	defer task.End()

	if err := app.listEntities(ctx, app.cfg.Output, app.cfg.ListFlags); err != nil {
		return err
	}

	return nil
}

func (app *App) runExport(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "runExport")
	defer task.End()

	if err := app.Export(ctx,
		app.cfg.ExportName,
	); err != nil {
		return err
	}

	return nil
}

func (app *App) runDump(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "runDump")
	defer task.End()

	n, err := app.dump(ctx, app.cfg.Input)
	if err != nil {
		return err
	}

	app.l().Printf("dumped %d item(s)", n)
	return nil
}

func (app *App) l() logger.Interface {
	// inherit the logger from the slackdump options
	if app.cfg.Options.Logger == nil {
		app.cfg.Options.Logger = logger.Default
	}
	return app.cfg.Options.Logger
}
