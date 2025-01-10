package export

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
)

// export runs the export v3.
func export(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, list *structures.EntityList, params exportFlags) error {
	lg := cfg.Log

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}

	lg.InfoContext(ctx, "temporary directory in use", "tmpdir", tmpdir)
	chunkdir, err := chunk.OpenDir(tmpdir)
	if err != nil {
		return err
	}
	defer chunkdir.Close()
	if !lg.Enabled(ctx, slog.LevelDebug) {
		defer func() { _ = chunkdir.RemoveAll() }()
	}
	updFn := func() func(_ *slack.Channel, m *slack.Message) error {
		// hack: wrapper around the message update function, which does not
		// have the channel parameter.  TODO: fix this in the library.
		fn := fileproc.ExportTokenUpdateFn(params.ExportToken)
		return func(_ *slack.Channel, m *slack.Message) error {
			return fn(m)
		}
	}
	conv := transform.NewExpConverter(chunkdir, fsa, transform.ExpWithMsgUpdateFunc(updFn()))
	tf := transform.NewExportCoordinator(ctx, conv, transform.WithBufferSize(1000))
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.DownloadFiles && params.ExportStorageType != fileproc.STnone
	sdl, stop := fileproc.NewDownloader(ctx, dlEnabled, sess.Client(), fsa, lg)
	defer stop()

	pb := bootstrap.ProgressBar(ctx, lg, progressbar.OptionShowCount()) // progress bar

	stream := sess.Stream(
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptResultFn(func(sr stream.Result) error {
			lg.DebugContext(ctx, "conversations", "sr", sr.String())
			pb.Describe(sr.String())
			_ = pb.Add(1)
			return nil
		}),
	)

	flags := control.Flags{
		MemberOnly:  cfg.MemberOnly,
		RecordFiles: false, // archive format is transitory, don't need extra info.
	}
	ctr := control.New(
		chunkdir,
		stream,
		control.WithFiler(fileproc.NewExport(params.ExportStorageType, sdl)),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithTransformer(tf),
	)

	lg.InfoContext(ctx, "running export...")
	if err := ctr.Run(ctx, list); err != nil {
		_ = pb.Finish()
		return err
	}
	_ = pb.Finish()
	// at this point no goroutines are running, we are safe to assume that
	// everything we need is in the chunk directory.
	if err := conv.WriteIndex(); err != nil {
		return err
	}
	if err := tf.Close(); err != nil {
		return err
	}
	pb.Describe("OK")
	lg.Debug("index written")
	lg.InfoContext(ctx, "conversations export finished")
	lg.DebugContext(ctx, "chunk files retained", "dir", tmpdir)
	return nil
}

// progresser is an interface for progress bars.
type progresser interface {
	RenderBlank() error
	Describe(description string)
	Add(num int) error
	Finish() error
}
