package app

import (
	"context"
	"errors"

	"github.com/rusq/slackdump/internal/export"
)

// Export performs the full export of slack workspace in slack export compatible
// format.
func (app *App) Export(ctx context.Context, dir string) error {
	if dir == "" { // dir is passed from app.cfg.ExportDirectory
		return errors.New("export directory not specified")
	}

	export := export.New(dir, app.sd)
	return export.Run(ctx)
}
