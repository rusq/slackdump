package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
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

// TimedSpinner displays a spinner for the duration of the operation.  It runs
// in a separate goroutine and stops once the stop function is called.
func TimedSpinner(ctx context.Context, w io.Writer, title string, max int, interval time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)

	var wait func()
	go func() {
		wait = fakeProgress(ctx, w, title, max, interval)
	}()
	return func() {
		cancel()
		wait()
	}
}

const defaultInterval = 500 * time.Millisecond

// fakeProgress starts a fake spinner and returns a channel that must be closed
// once the operation completes. interval is interval between iterations. If not
// set, will default to 100ms.
func fakeProgress(ctx context.Context, w io.Writer, title string, max int, interval time.Duration) (wait func()) {
	if cfg.Log.Enabled(ctx, slog.LevelDebug) {
		return func() {}
	}
	if interval == 0 {
		interval = defaultInterval
	}
	finished := make(chan struct{})
	go func() {
		bar := progressbar.NewOptions(
			max,
			progressbar.OptionSetDescription(title),
			progressbar.OptionSetPredictTime(false),
			progressbar.OptionSpinnerType(61),
			progressbar.OptionSetWriter(w),
		)
		t := time.NewTicker(interval)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				bar.Clear()
				bar.Finish()
				fmt.Fprintln(w)
				close(finished)
				return
			case <-t.C:
				bar.Add(1)
			}
		}
	}()
	return func() {
		<-finished
	}
}
