package export

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
	"github.com/rusq/slackdump/v3/internal/chunk/transform"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/source"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
)

// export runs the export v3.
func exportv31(ctx context.Context, sess *slackdump.Session, fsa fsadapter.FS, list *structures.EntityList, params exportFlags) error {
	lg := cfg.Log

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}

	lg.InfoContext(ctx, "temporary directory in use", "tmpdir", tmpdir)
	dbfile := filepath.Join(tmpdir, "slackdump.sqlite")
	// wconn is the writer connection
	wconn, err := sqlx.Open(repository.Driver, dbfile)
	if err != nil {
		return err
	}
	si := dbproc.SessionInfo{
		FromTS:         (*time.Time)(&cfg.Oldest),
		ToTS:           (*time.Time)(&cfg.Latest),
		FilesEnabled:   cfg.DownloadFiles,
		AvatarsEnabled: cfg.DownloadAvatars,
		Mode:           "export",
		Args:           strings.Join(os.Args, "|"),
	}

	tmpdbp, err := dbproc.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpdbp.Close(); err != nil {
			lg.ErrorContext(ctx, "unable to close database processor", "error", err)
		}
	}()
	src := source.OpenDatabaseConn(wconn)
	if !lg.Enabled(ctx, slog.LevelDebug) {
		defer func() { _ = os.RemoveAll(tmpdir) }()
	}

	conv := transform.NewExpConverter(src, fsa, transform.ExpWithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(params.ExportToken)))
	tf := transform.NewExportCoordinator(ctx, conv, transform.WithBufferSize(1000))
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.DownloadFiles && params.ExportStorageType != fileproc.STnone
	fdl := fileproc.NewDownloader(ctx, dlEnabled, sess.Client(), fsa, lg)
	fp := fileproc.NewExport(params.ExportStorageType, fdl)
	avdl := fileproc.NewDownloader(ctx, cfg.DownloadAvatars, sess.Client(), fsa, lg)
	avp := fileproc.NewAvatarProc(avdl)

	lg.InfoContext(ctx, "running export...")
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
	ctr, err := control.NewDB(
		ctx,
		stream,
		tmpdbp,
		control.WithFiler(fp),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithTransformer(tf),
		control.WithAvatarProcessor(avp),
	)
	if err != nil {
		return fmt.Errorf("error creating db controller: %w", err)
	}
	defer ctr.Close()

	if err := ctr.Run(ctx, list); err != nil {
		_ = pb.Finish()
		return err
	}
	_ = pb.Finish()
	// at this point no goroutines are running, we are safe to assume that
	// everything we need is in the chunk directory.
	if err := conv.WriteIndex(ctx); err != nil {
		return err
	}
	lg.Debug("index written")
	if err := tf.Close(); err != nil {
		return err
	}
	pb.Describe("OK")
	lg.InfoContext(ctx, "conversations export finished")
	lg.DebugContext(ctx, "chunk files retained", "dir", tmpdir)
	return nil
}

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
	src := source.NewChunkDir(chunkdir, true)
	conv := transform.NewExpConverter(src, fsa, transform.ExpWithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(params.ExportToken)))
	tf := transform.NewExportCoordinator(ctx, conv, transform.WithBufferSize(1000))
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.DownloadFiles && params.ExportStorageType != fileproc.STnone
	fdl := fileproc.NewDownloader(ctx, dlEnabled, sess.Client(), fsa, lg)
	fp := fileproc.NewExport(params.ExportStorageType, fdl)
	avdl := fileproc.NewDownloader(ctx, cfg.DownloadAvatars, sess.Client(), fsa, lg)
	avp := fileproc.NewAvatarProc(avdl)

	lg.InfoContext(ctx, "running export...")
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
	ctr := control.NewDir(
		chunkdir,
		stream,
		control.WithFiler(fp),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithTransformer(tf),
		control.WithAvatarProcessor(avp),
	)
	defer ctr.Close()

	if err := ctr.Run(ctx, list); err != nil {
		_ = pb.Finish()
		return err
	}
	_ = pb.Finish()
	// at this point no goroutines are running, we are safe to assume that
	// everything we need is in the chunk directory.
	if err := conv.WriteIndex(ctx); err != nil {
		return err
	}
	lg.Debug("index written")
	if err := tf.Close(); err != nil {
		return err
	}
	pb.Describe("OK")
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
