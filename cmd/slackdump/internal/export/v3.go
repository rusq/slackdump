package export

import (
	"context"
	"fmt"
	"os"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v2"

	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/control"
	"github.com/rusq/slackdump/v2/internal/chunk/transform"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
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
	if !lg.IsDebug() {
		defer chunkdir.RemoveAll()
	}
	tf, err := transform.NewExport(ctx, fsa, tmpdir, transform.WithBufferSize(1000), transform.WithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(params.ExportToken)))
	if err != nil {
		return fmt.Errorf("failed to create transformer: %w", err)
	}
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.DumpFiles && params.ExportStorageType != fileproc.STNone
	sdl, stop := fileproc.NewDownloader(ctx, dlEnabled, sess.Client(), fsa, lg)
	defer stop()

	pb := newProgressBar(progressbar.NewOptions(
		-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(8)),
		lg.IsDebug(),
	)
	pb.RenderBlank()

	flags := control.Flags{
		MemberOnly: params.MemberOnly,
	}
	stream := sess.Stream(
		slackdump.OptOldest(params.Oldest),
		slackdump.OptLatest(params.Latest),
		slackdump.OptResultFn(func(sr slackdump.StreamResult) error {
			lg.Debugf("conversations: %s", sr.String())
			pb.Describe(sr.String())
			pb.Add(1)
			return nil
		}),
	)
	ctr := control.New(
		chunkdir,
		stream,
		control.WithSubproc(fileproc.NewExport(params.ExportStorageType, sdl)),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithTransformer(tf),
	)

	lg.Print("running export...")
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
