package export

import (
	"context"
	"fmt"
	"os"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v2"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/downloader"
	"github.com/rusq/slackdump/v2/export"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/ctrl"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

func exportV3(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, list *structures.EntityList, options export.Config) error {
	lg := logger.FromContext(ctx)

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}
	chunkdir, err := chunk.OpenDir(tmpdir)
	if err != nil {
		return err
	}
	if !lg.IsDebug() {
		defer chunkdir.RemoveAll()
	}
	tf, err := transform.NewExport(ctx, fsa, tmpdir, transform.WithBufferSize(1000), transform.WithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(options.ExportToken)))
	if err != nil {
		return fmt.Errorf("failed to create transformer: %w", err)
	}
	defer tf.Close()

	// starting the downloader
	sdl, stop := initDownloader(ctx, cfg.DumpFiles, sess.Client(), options.Type, fsa, lg)
	defer stop()

	pb := newProgressBar(progressbar.NewOptions(
		-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(8)),
		lg.IsDebug(),
	)
	pb.RenderBlank()

	flags := ctrl.Flags{
		MemberOnly: options.MemberOnly,
	}
	ctr := ctrl.New(
		chunkdir,
		sess.Stream(),
		ctrl.WithFiler(fileproc.NewExport(options.Type, sdl)),
		ctrl.WithLogger(lg),
		ctrl.WithFlags(flags),
		ctrl.WithTransformer(tf),
		ctrl.WithResultFn(func(sr slackdump.StreamResult) error {
			lg.Debugf("conversations: %s", sr.String())
			pb.Describe(sr.String())
			pb.Add(1)
			return nil
		}),
	)

	if err := ctr.Run(ctx, list); err != nil {
		return err
	}
	pb.Finish()
	// at this point no goroutines are running, we are safe to assume that
	// everything we need is in the chunk directory.
	if err := tf.WriteIndex(); err != nil {
		return err
	}
	pb.Describe("OK")
	lg.Debug("index written")
	lg.Println("conversations export finished")
	lg.Debugf("chunk files in: %s", tmpdir)
	return nil
}

func initDownloader(ctx context.Context, gEnabled bool, cl downloader.Downloader, t export.ExportType, fsa fsadapter.FS, lg logger.Interface) (sdl fileproc.Downloader, stop func()) {
	if t == export.TNoDownload || !gEnabled {
		return fileproc.NoopDownloader{}, func() {}
	} else {
		dl := downloader.New(cl, fsa, downloader.WithLogger(lg))
		dl.Start(ctx)
		return dl, dl.Stop
	}
}

func newProgressBar(pb *progressbar.ProgressBar, debug bool) progresser {
	if debug {
		return progressbar.DefaultSilent(0)
	}
	return pb
}

// progresser is an interface for progress bars.
type progresser interface {
	RenderBlank() error
	Describe(description string)
	Add(num int) error
	Finish() error
}
