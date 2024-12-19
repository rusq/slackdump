package bootstrap

import (
	"context"
	"log/slog"

	"github.com/schollz/progressbar/v3"
)

func ProgressBar(ctx context.Context, lg *slog.Logger, opts ...progressbar.Option) *progressbar.ProgressBar {
	fullopts := append([]progressbar.Option{
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(8),
	}, opts...)

	pb := newProgressBar(progressbar.NewOptions(
		-1,
		fullopts...),
		lg.Enabled(ctx, slog.LevelDebug),
	)
	_ = pb.RenderBlank()
	return pb
}

func newProgressBar(pb *progressbar.ProgressBar, debug bool) *progressbar.ProgressBar {
	if debug {
		return progressbar.DefaultSilent(0)
	}
	return pb
}
