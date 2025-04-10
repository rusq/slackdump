package export

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/schollz/progressbar/v3"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/client"
	"github.com/rusq/slackdump/v3/internal/convert/transform"
	"github.com/rusq/slackdump/v3/internal/convert/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/source"
	"github.com/rusq/slackdump/v3/stream"
)

// export runs the export v3.1.
func exportv31(ctx context.Context, sess client.Slack, fsa fsadapter.FS, list *structures.EntityList, params exportFlags) error {
	lg := cfg.Log

	tmpdir, err := os.MkdirTemp("", "slackdump-*")
	if err != nil {
		return err
	}

	lg.InfoContext(ctx, "temporary directory in use", "tmpdir", tmpdir)
	wconn, si, err := bootstrap.Database(tmpdir, "export")
	if err != nil {
		return err
	}
	defer wconn.Close()

	tmpdbp, err := dbase.New(ctx, wconn, si)
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpdbp.Close(); err != nil {
			lg.ErrorContext(ctx, "unable to close database processor", "error", err)
		}
	}()
	src := source.DatabaseWithSource(tmpdbp.Source())
	if !lg.Enabled(ctx, slog.LevelDebug) {
		defer func() { _ = os.RemoveAll(tmpdir) }()
	}

	conv := transform.NewExpConverter(src, fsa, transform.ExpWithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(params.ExportToken)))
	tf := transform.NewExportCoordinator(ctx, conv, transform.WithBufferSize(1000))
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.WithFiles && params.ExportStorageType != source.STnone
	fdl := fileproc.NewDownloader(ctx, dlEnabled, sess, fsa, lg)
	fp := fileproc.NewExport(params.ExportStorageType, fdl)
	avdl := fileproc.NewDownloader(ctx, cfg.WithAvatars, sess, fsa, lg)
	avp := fileproc.NewAvatarProc(avdl)

	lg.InfoContext(ctx, "running export...")
	pb := bootstrap.ProgressBar(ctx, lg, progressbar.OptionShowCount()) // progress bar

	s := stream.New(sess, cfg.Limits,
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
		MemberOnly:   cfg.MemberOnly,
		RecordFiles:  false, // archive format is transitory, don't need extra info.
		ChannelUsers: cfg.OnlyChannelUsers,
	}
	ctr, err := control.New(
		ctx,
		s,
		tmpdbp,
		control.WithFiler(fp),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithCoordinator(tf),
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
//
// Deprecated: use exportv31 instead.
func export(ctx context.Context, sess client.Slack, fsa fsadapter.FS, list *structures.EntityList, params exportFlags) error {
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
	src := source.OpenChunkDir(chunkdir, true)
	conv := transform.NewExpConverter(src, fsa, transform.ExpWithMsgUpdateFunc(fileproc.ExportTokenUpdateFn(params.ExportToken)))
	tf := transform.NewExportCoordinator(ctx, conv, transform.WithBufferSize(1000))
	defer tf.Close()

	// starting the downloader
	dlEnabled := cfg.WithFiles && params.ExportStorageType != source.STnone
	fdl := fileproc.NewDownloader(ctx, dlEnabled, sess, fsa, lg)
	fp := fileproc.NewExport(params.ExportStorageType, fdl)
	avdl := fileproc.NewDownloader(ctx, cfg.WithAvatars, sess, fsa, lg)
	avp := fileproc.NewAvatarProc(avdl)

	lg.InfoContext(ctx, "running export...")
	pb := bootstrap.ProgressBar(ctx, lg, progressbar.OptionShowCount()) // progress bar

	stream := stream.New(sess, cfg.Limits,
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
		MemberOnly:   cfg.MemberOnly,
		RecordFiles:  false, // archive format is transitory, don't need extra info.
		ChannelUsers: cfg.OnlyChannelUsers,
	}
	ctr := control.NewDir(
		chunkdir,
		stream,
		control.WithFiler(fp),
		control.WithLogger(lg),
		control.WithFlags(flags),
		control.WithCoordinator(tf),
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
