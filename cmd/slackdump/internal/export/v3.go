package export

import (
	"context"
	"os"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slack"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/logger"
	"github.com/rusq/slackdump/v3/stream"
)

// TODO: check if the features is on par with Export v2.

// exportV3 runs the export v3.
func exportV3(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, list *structures.EntityList, params exportFlags) error {
	lg := logger.FromContext(ctx)

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}

	lg.Printf("using %s as the temporary directory", tmpdir)
	chunkdir, err := chunk.OpenDir(tmpdir)
	if err != nil {
		return err
	}
	defer chunkdir.Close()
	if !lg.IsDebug() {
		defer chunkdir.RemoveAll()
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

	pb := newProgressBar(progressbar.NewOptions(
		-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(8)),
		lg.IsDebug(),
	)
	pb.RenderBlank()

	stream := sess.Stream(
		stream.OptOldest(params.Oldest),
		stream.OptLatest(params.Latest),
		stream.OptResultFn(func(sr stream.StreamResult) error {
			lg.Debugf("conversations: %s", sr.String())
			pb.Describe(sr.String())
			pb.Add(1)
			return nil
		}),
	)

	flags := control.Flags{
		MemberOnly: params.MemberOnly,
	}
	ctr := control.New(
		chunkdir,
		stream,
		control.WithFiler(fileproc.NewExport(params.ExportStorageType, sdl)),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithTransformer(tf),
	)

	lg.Print("running export...")
	if err := ctr.Run(ctx, list); err != nil {
		pb.Finish()
		return err
	}
	pb.Finish()
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
	lg.Println("conversations export finished")
	lg.Debugf("chunk files in: %s", tmpdir)
	return nil
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
