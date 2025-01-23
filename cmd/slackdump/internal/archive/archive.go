package archive

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"time"

	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/bootstrap"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v3/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/control"
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
	start := time.Now()
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}

	cfg.Output = cfg.StripZipExt(cfg.Output)
	if cfg.Output == "" {
		base.SetExitStatus(base.SInvalidParameters)
		return errNoOutput
	}

	cd, err := chunk.CreateDir(cfg.Output)
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}

	sess, err := bootstrap.SlackdumpSession(ctx)
	if err != nil {
		base.SetExitStatus(base.SInitializationError)
		return err
	}

	lg := cfg.Log
	// start attachment downloader
	dl, stop := fileproc.NewDownloader(
		ctx,
		cfg.DownloadFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	defer stop()
	subproc := fileproc.NewExport(fileproc.STmattermost, dl)
	// start avatar downloader
	avdl, avstop := fileproc.NewDownloader(
		ctx,
		cfg.DownloadAvatars,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	defer avstop()
	avproc := fileproc.NewAvatarProc(avdl)

	stream := sess.Stream(
		stream.OptLatest(time.Time(cfg.Latest)),
		stream.OptOldest(time.Time(cfg.Oldest)),
		stream.OptResultFn(resultLogger(lg)),
	)
	ctrl := control.New(
		cd,
		stream,
		control.WithLogger(lg),
		control.WithFiler(subproc),
		control.WithFlags(control.Flags{MemberOnly: cfg.MemberOnly, RecordFiles: cfg.RecordFiles}),
		control.WithAvatarProcessor(avproc),
	)
	if err := ctrl.Run(ctx, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	lg.Info("Recorded workspace data", "filename", cd.Name(), "took", time.Since(start))

	return nil
}

func resultLogger(lg *slog.Logger) func(sr stream.Result) error {
	return func(sr stream.Result) error {
		lg.Info("stream", "result", sr.String())
		return nil
	}
}
