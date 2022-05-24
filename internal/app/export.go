package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rusq/slackdump/v2/fsadapter"
	"github.com/rusq/slackdump/v2/internal/export"
)

// Export performs the full export of slack workspace in slack export compatible
// format.
func (app *App) Export(ctx context.Context, dir string) error {
	if dir == "" { // dir is passed from app.cfg.ExportDirectory
		return errors.New("export directory not specified")
	}

	cfg := export.Options{
		Oldest:       time.Time(app.cfg.Oldest),
		Latest:       time.Time(app.cfg.Latest),
		IncludeFiles: app.cfg.Options.DumpFiles,
	}
	zf, err := fsadapter.NewZipFile(strings.TrimSuffix(dir, ".zip") + ".zip") // ensure we have a zip suffix TODO
	if err != nil {
		return err
	}
	defer zf.Close()
	export := export.New(app.sd, zf, cfg)
	return export.Run(ctx)
}
