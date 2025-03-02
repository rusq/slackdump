package archive

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc"
	"github.com/rusq/slackdump/v3/internal/chunk/dbproc/repository"
	"github.com/rusq/slackdump/v3/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/stream"
)

//go:embed assets/archive.md
var mdArchive string

var CmdArchive = &base.Command{
	Run:         RunArchive,
	UsageLine:   "slackdump archive [flags] [link1[ link 2[ link N]]]",
	Short:       "archive the workspace or individual conversations on disk",
	Long:        mdArchive,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	RequireAuth: true,
	PrintFlags:  true,
}

func init() {
	CmdArchive.Wizard = archiveWizard
}

var errNoOutput = errors.New("output directory is required")

func RunArchive(ctx context.Context, cmd *base.Command, args []string) error {
	if cfg.UseChunkFiles {
		return runChunkArchive(ctx, cmd, args)
	} else {
		return runDBArchive(ctx, cmd, args)
	}
}

func runChunkArchive(ctx context.Context, _ *base.Command, args []string) error {
	start := time.Now()
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}
	cd, err := NewDirectory(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	defer cd.Close()

	ctrl, err := ArchiveController(ctx, cd, sess)
	if err != nil {
		return err
	}
	defer ctrl.Close()
	if err := ctrl.Run(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	cfg.Log.Info("Recorded workspace data", "directory", cd.Name(), "took", time.Since(start))
	return nil
}

func runDBArchive(ctx context.Context, cmd *base.Command, args []string) error {
	start := time.Now()
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	dirname := cfg.StripZipExt(cfg.Output)
	if err := os.MkdirAll(dirname, 0o755); err != nil {
		return err
	}

	conn, err := sqlx.Open(repository.Driver, filepath.Join(dirname, "slackdump.sqlite"))
	if err != nil {
		return err
	}
	defer conn.Close()

	ctrl, err := DBController(ctx, cmd, conn, sess, dirname)
	if err != nil {
		return err
	}

	defer func() {
		if err := ctrl.Close(); err != nil {
			slog.ErrorContext(ctx, "unable to close database controller", "error", err)
		}
	}()

	if err := ctrl.Run(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	cfg.Log.Info("Recorded workspace data", "directory", dirname, "took", time.Since(start))

	return nil
}

// NewDirectory creates a new chunk directory with name.  If name has a .zip
// extension it is stripped.
func NewDirectory(name string) (*chunk.Directory, error) {
	name = cfg.StripZipExt(name)
	if name == "" {
		return nil, errNoOutput
	}

	cd, err := chunk.CreateDir(name)
	if err != nil {
		return nil, err
	}
	return cd, nil
}

func DBController(ctx context.Context, cmd *base.Command, conn *sqlx.DB, sess *slackdump.Session, dirname string, opts ...stream.Option) (RunCloser, error) {
	lg := cfg.Log
	dbp, err := dbproc.New(ctx, conn, bootstrap.SessionInfo(cmd.Name()))
	if err != nil {
		return nil, err
	}
	sopts := []stream.Option{
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	}
	sopts = append(sopts, opts...)
	// start attachment downloader
	dl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(dirname),
		lg,
	)
	// start avatar downloader
	avdl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadAvatars,
		sess.Client(),
		fsadapter.NewDirectory(dirname),
		lg,
	)

	ctrl, err := control.NewDB(
		ctx,
		sess.Stream(sopts...),
		dbp,
		control.WithFiler(fileproc.New(dl)),
		control.WithAvatarProcessor(fileproc.NewAvatarProc(avdl)),
	)
	if err != nil {
		return nil, err
	}
	return ctrl, nil
}

type RunCloser interface {
	Run(context.Context, *structures.EntityList) error
	io.Closer
}

// ArchiveController returns the default archive controller initialised based
// on global configuration parameters.
func ArchiveController(ctx context.Context, cd *chunk.Directory, sess *slackdump.Session, opts ...stream.Option) (*control.DirController, error) {
	lg := cfg.Log

	sopts := []stream.Option{
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	}
	sopts = append(sopts, opts...)

	// start attachment downloader
	dl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	// start avatar downloader
	avdl := fileproc.NewDownloader(
		ctx,
		cfg.DownloadAvatars,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)

	ctrl := control.NewDir(
		cd,
		sess.Stream(sopts...),
		control.WithLogger(lg),
		control.WithFlags(control.Flags{MemberOnly: cfg.MemberOnly, RecordFiles: cfg.RecordFiles}),
		control.WithFiler(fileproc.New(dl)),
		control.WithAvatarProcessor(fileproc.NewAvatarProc(avdl)),
	)
	return ctrl, nil
}

func resultLogger(lg *slog.Logger) func(sr stream.Result) error {
	return func(sr stream.Result) error {
		lg.Info("stream", "result", sr.String())
		return nil
	}
}
