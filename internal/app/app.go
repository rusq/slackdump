package app

import (
	"context"
	"runtime/trace"
	"time"

	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/app/config"
	"github.com/rusq/slackdump/v2/internal/app/emoji"
)

// Run starts the Slackdump.
func Run(ctx context.Context, cfg config.Params, prov auth.Provider) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	ctx, task := trace.NewTask(ctx, "Run")
	defer task.End()

	start := time.Now()

	var err error
	if cfg.ExportName != "" {
		err = Export(ctx, cfg, prov)
	} else if cfg.Emoji.Enabled {
		err = emoji.Download(ctx, cfg, prov)
	} else {
		err = Dump(ctx, cfg, prov)
	}
	if err != nil {
		return err
	}

	cfg.Logger().Printf("completed, time taken: %s", time.Since(start))
	return nil
}
