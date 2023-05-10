package record

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/rusq/fsadapter"
	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/control"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

//go:embed assets/record.md
var mdRecord string

var CmdRecord = &base.Command{
	Run:         RunRecord,
	UsageLine:   "slackdump record [link1[ link 2[ link N]]]",
	Short:       "record the dump of the workspace or individual conversations",
	Long:        mdRecord,
	FlagMask:    cfg.OmitUserCacheFlag | cfg.OmitCacheDir,
	RequireAuth: true,
	PrintFlags:  true,
}

func RunRecord(ctx context.Context, cmd *base.Command, args []string) error {
	list, err := structures.NewEntityList(args)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		base.SetExitStatus(base.SAuthError)
		return err
	}

	// hack
	cfg.BaseLocation = strings.TrimSuffix(cfg.BaseLocation, ".zip")

	cd, err := chunk.CreateDir(cfg.BaseLocation)
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}

	sess, err := slackdump.New(ctx, prov, slackdump.WithLogger(logger.FromContext(ctx)))
	if err != nil {
		base.SetExitStatus(base.SGenericError)
		return err
	}

	lg := logger.FromContext(ctx)
	stream := sess.Stream(
		slackdump.OptLatest(time.Time(cfg.Latest)),
		slackdump.OptOldest(time.Time(cfg.Oldest)),
		slackdump.OptResultFn(resultLogger(lg)),
	)
	dl, stop := fileproc.NewDownloader(
		ctx,
		cfg.DumpFiles,
		sess.Client(),
		fsadapter.NewDirectory(cd.Name()),
		lg,
	)
	defer stop()
	// we are using the same file subprocessor as the mattermost export.
	subproc := fileproc.NewExport(fileproc.STmattermost, dl)
	ctrl := control.New(cd, stream, control.WithLogger(lg), control.WithSubproc(subproc))
	if err := ctrl.Run(ctx, list); err != nil {
		return err
	}
	lg.Printf("Recorded workspace data to %s", cd.Name())

	return nil
}

func resultLogger(lg logger.Interface) func(sr slackdump.StreamResult) error {
	return func(sr slackdump.StreamResult) error {
		lg.Printf("%s", sr)
		return nil
	}
}
