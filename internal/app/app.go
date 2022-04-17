package app

import (
	"context"
	"html/template"
	"runtime/trace"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/slackdump/v2"
)

const (
	OutputTypeJSON = "json"
	OutputTypeText = "text"
)

type App struct {
	sd   *slackdump.SlackDumper
	tmpl *template.Template

	cfg Config
}

// New creates a new slackdump app.
func New(cfg Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	tmpl, err := cfg.compileTemplates()
	if err != nil {
		return nil, err
	}
	return &App{cfg: cfg, tmpl: tmpl}, nil
}

func (app *App) Run(ctx context.Context) error {
	ctx, task := trace.NewTask(ctx, "app.Run")
	defer task.End()

	if err := app.init(ctx); err != nil {
		return err
	}

	start := time.Now()
	var err error
	switch {
	case app.cfg.ListFlags.FlagsPresent():
		err = app.runListEntities(ctx)
	case app.cfg.FullExport:
		err = app.runExport(ctx)
	default:
		err = app.runDump(ctx)
	}

	if err != nil {
		trace.Log(ctx, "error", err.Error())
		return err
	}

	dlog.Printf("completed, time taken: %s", time.Since(start))
	return nil
}

// init initialises the slack dumper app.
func (app *App) init(ctx context.Context) error {
	sd, err := slackdump.NewWithOptions(
		ctx,
		app.cfg.Creds.Token,
		app.cfg.Creds.Cookie,
		app.cfg.Options,
	)
	if err != nil {
		return err
	}
	app.sd = sd
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
		app.cfg.ExportDirectory,
		time.Time(app.cfg.Oldest),
		time.Time(app.cfg.Latest),
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

	dlog.Printf("dumped %d item(s)", n)
	return nil
}
