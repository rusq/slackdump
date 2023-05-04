package export

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rusq/dlog"
	"github.com/rusq/fsadapter"

	"github.com/rusq/slackdump/v2"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/cfg"
	"github.com/rusq/slackdump/v2/cmd/slackdump/internal/golang/base"
	"github.com/rusq/slackdump/v2/internal/chunk/transform/fileproc"
	"github.com/rusq/slackdump/v2/internal/structures"
	"github.com/rusq/slackdump/v2/logger"
)

var CmdExport = &base.Command{
	Run:         nil,
	Wizard:      nil,
	UsageLine:   "slackdump export",
	Short:       "exports the Slack Workspace or individual conversations",
	FlagMask:    cfg.OmitUserCacheFlag,
	Long:        ``, // TODO: add long description
	CustomFlags: false,
	PrintFlags:  true,
	RequireAuth: true,
}

type exportFlags struct {
	ExportStorageType fileproc.StorageType
	MemberOnly        bool
	ExportToken       string
	Oldest            time.Time
	Latest            time.Time
}

var (
	compat  bool
	options = exportFlags{
		ExportStorageType: fileproc.STMattermost,
		Oldest:            time.Time(cfg.Oldest),
		Latest:            time.Time(cfg.Latest),
	}
)

func init() {
	// TODO: move TimeValue somewhere more appropriate once v1 is sunset.
	CmdExport.Flag.Var(&options.ExportStorageType, "type", "export file storage type")
	CmdExport.Flag.BoolVar(&options.MemberOnly, "member-only", false, "export only channels, which current user belongs to")
	CmdExport.Flag.StringVar(&options.ExportToken, "export-token", "", "file export token to append to each of the file URLs")
	CmdExport.Flag.BoolVar(&compat, "compat", false, "use the v2 export code")

	CmdExport.Run = runExport
	CmdExport.Wizard = wizExport
}

func runExport(ctx context.Context, cmd *base.Command, args []string) error {
	start := time.Now()
	if cfg.BaseLocation == "" {
		return errors.New("use -base to set the base output location")
	}
	if !cfg.DumpFiles {
		options.ExportStorageType = fileproc.STNone
	}
	list, err := structures.NewEntityList(args)
	if err != nil {
		return fmt.Errorf("error parsing the entity list: %w", err)
	}

	prov, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	sess, err := slackdump.New(ctx, prov, slackdump.WithLogger(logger.FromContext(ctx)), slackdump.WithLimits(cfg.Limits))
	if err != nil {
		return err
	}

	fsa, err := fsadapter.New(cfg.BaseLocation)
	if err != nil {
		base.SetExitStatus(base.SUserError)
		return err
	}
	defer func() {
		dlog.Debugln("closing the fsadapter")
		fsa.Close()
	}()

	var expfn = exportV3
	if compat {
		expfn = exportV2
	}

	if err := expfn(ctx, sess, fsa, list, options); err != nil {
		base.SetExitStatus(base.SApplicationError)
		return fmt.Errorf("export failed: %w", err)
	}

	lg := logger.FromContext(ctx)
	lg.Printf("export completed in %s", time.Since(start).Truncate(time.Second).String())
	return nil
}
