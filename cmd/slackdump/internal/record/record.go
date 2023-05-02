package record

import (
	"context"
	_ "embed"
	"strings"
	"time"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/chunk"
	"github.com/rusq/slackdump/v2/internal/chunk/control"
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

	stream := sess.Stream(slackdump.OptLatest(time.Time(cfg.Latest)), slackdump.OptOldest(time.Time(cfg.Oldest)))

	if err := record(ctx, stream, cd, list); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return err
	}
	return nil
}

func resultLogger(lg logger.Interface) func(sr slackdump.StreamResult) error {
	return func(sr slackdump.StreamResult) error {
		lg.Printf("%s", sr)
		return nil
	}
}

func record(ctx context.Context, stream control.Streamer, cd *chunk.Directory, list *structures.EntityList) error {
	lg := logger.FromContext(ctx)
	ctrl := control.New(cd, stream, control.WithLogger(lg), control.WithResultFn(resultLogger(lg)))
	if err := ctrl.Run(ctx, list); err != nil {
		return err
	}
	return nil
}
